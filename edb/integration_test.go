// +build integration

package edb

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/era/era"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

var exe = flag.String("e", "", "EDB executable")
var showEdbOutput = flag.Bool("show-edb-output", false, "")
var addrAPI, addrDB string

func TestMain(m *testing.M) {
	flag.Parse()
	if *exe == "" {
		log.Fatalln("You must provide the path of the EDB executable using th -e flag.")
	}
	if _, err := os.Stat(*exe); err != nil {
		log.Fatalln(err)
	}

	// get unused ports
	var listenerAPI, listenerDB net.Listener
	listenerAPI, addrAPI = getListenerAndAddr()
	listenerDB, addrDB = getListenerAndAddr()
	listenerAPI.Close()
	listenerDB.Close()

	os.Exit(m.Run())
}

func getListenerAndAddr() (net.Listener, string) {
	const localhost = "localhost:"

	listener, err := net.Listen("tcp", localhost)
	if err != nil {
		panic(err)
	}

	addr := listener.Addr().String()

	// addr contains IP address, we want hostname
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	return listener, localhost + port
}

// sanity test of the integration test environment
func TestTest(t *testing.T) {
	assert := assert.New(t)
	cfgFilename := createConfig()
	defer cleanupConfig(cfgFilename)
	assert.Nil(startEDB(cfgFilename).Kill())
}

func TestReaderWriter(t *testing.T) {
	assert := assert.New(t)

	caCert, caKey := createCertificate("Owner CA", "", "")
	readerCert, readerKey := createCertificate("Reader", caCert, caKey)
	writerCert, writerKey := createCertificate("Writer", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER reader REQUIRE ISSUER '/CN=Owner CA' SUBJECT '/CN=Reader'",
		"CREATE USER writer REQUIRE ISSUER '/CN=Owner CA' SUBJECT '/CN=Writer'",
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
		"GRANT SELECT ON test.data TO reader",
		"GRANT INSERT ON test.data TO writer",
	})

	cfgFilename := createConfig()
	defer cleanupConfig(cfgFilename)
	process := startEDB(cfgFilename)
	assert.NotNil(process)
	defer process.Kill()

	// Owner
	{
		serverCert := getServerCertificate()
		assert.Nil(postManifest(serverCert, manifest))
	}

	// Writer
	{
		serverCert := getServerCertificate()
		sig := getManifestSignature(serverCert)
		assert.Equal(calculateManifestSignature(manifest), sig)

		db := sqlOpen("writer", writerCert, writerKey, serverCert)
		_, err := db.Exec("INSERT INTO test.data VALUES (2), (6)")
		db.Close()
		assert.Nil(err)
	}

	// Reader
	{
		serverCert := getServerCertificate()
		sig := getManifestSignature(serverCert)
		assert.Equal(calculateManifestSignature(manifest), sig)

		var avg float64
		db := sqlOpen("reader", readerCert, readerKey, serverCert)
		assert.Nil(db.QueryRow("SELECT AVG(i) FROM test.data").Scan(&avg))
		_, err := db.Exec("INSERT INTO test.data VALUES (3)")
		db.Close()
		assert.NotNil(err)
		assert.Equal(4., avg)
	}
}

func TestPersistence(t *testing.T) {
	assert := assert.New(t)

	caCert, caKey := createCertificate("ca", "", "")
	usrCert, usrKey := createCertificate("usr", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
		"GRANT ALL ON test.data TO usr",
	})

	cfgFilename := createConfig()
	defer cleanupConfig(cfgFilename)

	process := startEDB(cfgFilename)
	assert.NotNil(process)

	serverCert := getServerCertificate()
	assert.Nil(postManifest(serverCert, manifest))

	db := sqlOpen("usr", usrCert, usrKey, serverCert)
	_, err := db.Exec("INSERT INTO test.data VALUES (2)")
	db.Close()
	assert.Nil(err)

	assert.Nil(process.Kill())

	// TODO: Find out why restarting EDB here sometimes fails (stdout/err seems to be empty)
	// TODO AB#875 This is from legacy TiDB-based EDB. Check if this is still true for MariaDB-based EDB.
	for i := 0; i < 3; i++ {
		process = startEDB(cfgFilename)
		if process != nil {
			break
		}
		log.Printf("TestPersistence: restart failed, trying again (%v/3)\n", i+1)
	}

	assert.NotNil(process)
	defer process.Kill()

	var val float64
	db = sqlOpen("usr", usrCert, usrKey, serverCert)
	assert.Nil(db.QueryRow("SELECT i FROM test.data").Scan(&val))
	db.Close()
	assert.Equal(2., val)
}

func DisabledTestInvalidQueryInManifest(t *testing.T) { //TODO
	assert := assert.New(t)

	cfgFilename := createConfig()
	defer cleanupConfig(cfgFilename)

	process := startEDB(cfgFilename)
	assert.NotNil(process)

	serverCert := getServerCertificate()

	assert.NotNil(postManifest(serverCert, createManifest("", []string{
		"CREATE TABL test.data (i INT)",
	})))

	// DB cannot be initialized after failed attempt
	assert.NotNil(postManifest(serverCert, createManifest("", []string{
		"CREATE TABLE test.data (i INT)",
	})))

	assert.Nil(process.Kill())

	// DB cannot be started after failed attempt
	log.SetOutput(ioutil.Discard)
	assert.Nil(startEDB(cfgFilename))
	log.SetOutput(os.Stdout)
}

func TestCurl(t *testing.T) {
	assert := assert.New(t)

	cfgFilename := createConfig()
	defer cleanupConfig(cfgFilename)
	process := startEDB(cfgFilename)
	assert.NotNil(process)
	defer process.Kill()

	cert := getServerCertificate()

	// Write certificate to temp file.
	certFile, err := ioutil.TempFile("", "")
	assert.Nil(err)
	certFilename := certFile.Name()
	_, err = certFile.WriteString(cert)
	certFile.Close()
	defer os.Remove(certFilename)
	assert.Nil(err)

	assert.Nil(exec.Command("curl", "--cacert", certFilename, "https://"+addrAPI+"/signature").Run())
}

func createConfig() string {
	cfg := core.Config{DatabaseAddress: addrDB, APIAddress: addrAPI}
	var err error
	cfg.DataPath, err = ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}

	file, err := ioutil.TempFile("", "")
	if err != nil {
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}

	name := file.Name()

	_, err = file.Write(jsonCfg)
	file.Close()
	if err != nil {
		os.Remove(name)
		os.RemoveAll(cfg.DataPath)
		panic(err)
	}

	return name
}

func cleanupConfig(filename string) {
	jsonCfg, err := ioutil.ReadFile(filename)
	os.Remove(filename)
	if err != nil {
		panic(err)
	}
	var cfg core.Config
	if err := json.Unmarshal(jsonCfg, &cfg); err != nil {
		panic(err)
	}
	if err := os.RemoveAll(cfg.DataPath); err != nil {
		panic(err)
	}
}

func startEDB(configFilename string) *os.Process {
	cmd := exec.Command(*exe, "-c", configFilename)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,            // group child with grandchildren so that we can kill 'em all
		Pdeathsig: syscall.SIGKILL, // kill child if test dies
	}
	go func() {
		if *showEdbOutput {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
			if err := cmd.Run(); isUnexpectedEDBError(err) {
				panic(err)
			}
		} else if out, err := cmd.CombinedOutput(); isUnexpectedEDBError(err) {
			log.Println("edb output:\n\n" + string(out))
			panic(err)
		}
	}()

	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "signature"}

	log.Println("EDB starting ...")
	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := client.Head(url.String())
		if err == nil {
			log.Println("EDB started")
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				panic(resp.Status)
			}
			cmd.Process.Pid *= -1 // let the Process object refer to the child process group
			return cmd.Process
		}
	}
}

func isUnexpectedEDBError(err error) bool {
	return err != nil && err.Error() != "signal: killed"
}

func createCertificate(commonName, signerCert, signerKey string) (cert, key string) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: commonName},
		NotAfter:     time.Now().Add(time.Hour),
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	var certBytes []byte

	if signerCert == "" {
		template.BasicConstraintsValid = true
		template.IsCA = true
		certBytes, _ = x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	} else {
		signer, _ := tls.X509KeyPair([]byte(signerCert), []byte(signerKey))
		parsedSignerCert, _ := x509.ParseCertificate(signer.Certificate[0])
		certBytes, _ = x509.CreateCertificate(rand.Reader, template, parsedSignerCert, &priv.PublicKey, signer.PrivateKey)
	}

	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, _ := x509.MarshalPKCS8PrivateKey(priv)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return string(pemCert), string(pemKey)
}

func getServerCertificate() string {
	blocks, err := era.InsecureGetCertificate(addrAPI)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(blocks[0]))
}

func createManifest(ca string, sql []string) []byte {
	manifest := struct {
		SQL []string
		CA  string
	}{sql, ca}
	jsonManifest, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	return jsonManifest
}

func calculateManifestSignature(manifest []byte) string {
	hash := sha256.Sum256(manifest)
	return hex.EncodeToString(hash[:])
}

func getManifestSignature(serverCert string) string {
	client := createHttpClient(serverCert)
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "signature"}

	resp, err := client.Get(url.String())
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(body)
}

func postManifest(serverCert string, manifest []byte) error {
	client := createHttpClient(serverCert)
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "manifest"}

	log.Print("posting manifest ...")
	resp, err := client.Post(url.String(), "", bytes.NewReader(manifest))
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	// wait until edb restarted
	url.Path = "signature"
	log.Print("waiting for restart ...")
	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := client.Get(url.String())
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				panic(err)
			}
			if resp.StatusCode != http.StatusOK {
				panic(resp.Status)
			}
			if len(body) > 0 {
				log.Print("restarted successfully")
				return nil
			}
		}
	}
}

func createHttpClient(serverCert string) http.Client {
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM([]byte(serverCert)); !ok {
		panic("AppendCertsFromPEM failed")
	}
	return http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

func sqlOpen(user, userCert, userKey, serverCert string) *sql.DB {
	cert, err := tls.X509KeyPair([]byte(userCert), []byte(userKey))
	if err != nil {
		panic(err)
	}
	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      x509.NewCertPool(),
	}
	if ok := tlsCfg.RootCAs.AppendCertsFromPEM([]byte(serverCert)); !ok {
		panic("AppendCertsFromPEM failed")
	}

	mysql.RegisterTLSConfig("custom", &tlsCfg)
	db, err := sql.Open("mysql", user+"@tcp("+addrDB+")/?tls=custom")
	if err != nil {
		panic(err)
	}
	return db
}

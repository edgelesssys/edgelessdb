//go:build integration
// +build integration

/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

package edb

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/base64"
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
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/edgelesssys/edgelessdb/edb/core"
	"github.com/edgelesssys/edgelessdb/edb/db"
	"github.com/edgelesssys/ego/marble"
	"github.com/edgelesssys/era/era"
	"github.com/edgelesssys/marblerun/coordinator/rpc"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	exe                = flag.String("e", "", "EDB executable")
	showEdbOutput      = flag.Bool("show-edb-output", false, "")
	attestationConfig  = flag.String("sgx-config", "", "Path to SGX config (containing UniqueID or triplet of SignerID, ProductID and SecurityVersion) to attestate against. Required to enable DCAP attestation testing.")
	attestationEnabled = os.Getenv("DCAP_TEST_ENABLED") == "1"
	addrAPI, addrDB    string
	coordinatorAddress string // For Marblerun integration tests
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *exe == "" {
		log.Fatalln("You must provide the path of the EDB executable using the -e flag.")
	}
	if _, err := os.Stat(*exe); err != nil {
		log.Fatalln(err)
	}
	if attestationEnabled && *attestationConfig != "" {
		log.Println("Testing with DCAP attestation enabled.")
		if _, err := os.Stat(*attestationConfig); err != nil {
			log.Fatalln(err)
		}
	} else {
		log.Println("Testing with DCAP attestation disabled.")
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
	setConfig(false, "")
	defer cleanupConfig()
	assert.Nil(startEDB("").Kill())
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
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
	assert.NotNil(process)
	defer process.Kill()

	// Owner
	{
		serverCert := getServerCertificate()
		_, err := postManifest(serverCert, manifest, true)
		assert.NoError(err)
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
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()

	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()
	_, err := postManifest(serverCert, manifest, true)
	assert.Nil(err)

	db := sqlOpen("usr", usrCert, usrKey, serverCert)
	_, err = db.Exec("INSERT INTO test.data VALUES (2)")
	db.Close()
	assert.Nil(err)

	assert.Nil(process.Kill())

	// Restart
	process = startEDB("")
	assert.NotNil(process, "restart failed!")
	defer process.Kill()

	var val float64
	db = sqlOpen("usr", usrCert, usrKey, serverCert)
	assert.Nil(db.QueryRow("SELECT i FROM test.data").Scan(&val))
	db.Close()
	assert.Equal(2., val)
}

func TestPersistenceEmptyDatabase(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	caCert, caKey := createCertificate("ca", "", "")
	usrCert, usrKey := createCertificate("usr", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"GRANT ALL ON *.* TO usr",
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()

	process := startEDB("")
	require.NotNil(process)

	serverCert := getServerCertificate()
	_, err := postManifest(serverCert, manifest, true)
	assert.Nil(err)

	db := sqlOpen("usr", usrCert, usrKey, serverCert)
	// regression: database creation wasn't persistently stored if it was the last action before ending the db process
	_, err = db.Exec("CREATE DATABASE emptydb")
	assert.Nil(err)
	db.Close()

	require.NoError(process.Kill())

	// Restart
	process = startEDB("")
	require.NotNil(process, "restart failed!")
	defer process.Kill()

	db = sqlOpen("usr", usrCert, usrKey, serverCert)
	defer db.Close()
	rows, err := db.Query("SHOW DATABASES like 'emptydb'")
	require.NoError(err)
	require.True(rows.Next())
	assert.False(rows.Next())
}

func TestAlterTable(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	caCert, caKey := createCertificate("ca", "", "")
	usrCert, usrKey := createCertificate("usr", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"CREATE DATABASE test",
		"GRANT ALL ON test.* TO usr",
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
	require.NotNil(process)
	defer process.Kill()

	serverCert := getServerCertificate()
	_, err := postManifest(serverCert, manifest, true)
	require.NoError(err)

	db := sqlOpen("usr", usrCert, usrKey, serverCert)
	defer db.Close()

	// https://github.com/edgelesssys/edgelessdb/issues/93
	_, err = db.Exec("CREATE TABLE test.data (i INT)")
	require.NoError(err)
	_, err = db.Exec("INSERT INTO test.data VALUES (2)")
	require.NoError(err)
	_, err = db.Exec("ALTER TABLE test.data ADD INDEX (i)")
	assert.NoError(err)
}

func TestManyConnections(t *testing.T) {
	require := require.New(t)

	caCert, caKey := createCertificate("ca", "", "")
	usrCert, usrKey := createCertificate("usr", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
	require.NotNil(process)
	defer process.Kill()

	serverCert := getServerCertificate()
	_, err := postManifest(serverCert, manifest, true)
	require.NoError(err)

	for i := 0; i < 150; i++ {
		db := sqlOpen("usr", usrCert, usrKey, serverCert)
		defer db.Close()
		require.NoError(db.Ping())
	}
}

func TestMisc(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	caCert, caKey := createCertificate("ca", "", "")
	usrCert, usrKey := createCertificate("usr", caCert, caKey)

	manifest := createManifest(caCert, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"GRANT ALL ON *.* TO usr",
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
	require.NotNil(process)
	defer process.Kill()

	serverCert := getServerCertificate()
	_, err := postManifest(serverCert, manifest, true)
	require.NoError(err)

	db := sqlOpen("usr", usrCert, usrKey, serverCert)
	defer db.Close()

	// charset is utf8mb4
	var name, charset string
	require.NoError(db.QueryRow("SHOW VARIABLES LIKE 'character_set_server'").Scan(&name, &charset))
	assert.Equal("utf8mb4", charset)

	// create table in nonexistent db fails
	_, err = db.Exec("CREATE TABLE test.data (i INT)")
	assert.Error(err)
}

func TestInvalidQueryInManifest(t *testing.T) {
	assert := assert.New(t)

	setConfig(false, "")
	defer cleanupConfig()

	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()

	_, err := postManifest(serverCert, createManifest("", []string{
		"CREATE TABL test.data (i INT)",
	}, false, ""), true)
	assert.Error(err)

	// DB cannot be initialized after failed attempt
	_, err = postManifest(serverCert, createManifest("", []string{
		"CREATE TABLE test.data (i INT)",
	}, false, ""), true)
	assert.Error(err)

	assert.Nil(process.Kill())

	// DB cannot be started after failed attempt
	log.SetOutput(ioutil.Discard)
	assert.Error(createEdbCmd("").Run())
	log.SetOutput(os.Stdout)
}

func TestCurl(t *testing.T) {
	assert := assert.New(t)

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
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

func TestMarbleReaderWriter(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Setup manifest
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
	}, true, "")

	// Setup mock Marblerun Coordinator
	grpcServer, tempDir, err := startMockMarblerunCoordinator(manifest)
	require.NoError(err)
	defer grpcServer.Stop()
	defer os.RemoveAll(tempDir)

	// Setup UUID dir
	marbleUUIDDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(marbleUUIDDir)

	// No SetConfig before launching edb here.
	// The marbleServer Activate function handles this part to mock Marblerun behavior.
	process := startEDB(marbleUUIDDir)
	assert.NotNil(process)
	defer process.Kill()

	// Wait until edb automatically restarts from "manifest from file deployment"
	// In theory, we could try to race for the server certificate and use the "secure" HTTP client here, however that seems like a bad idea for a test
	waitUntilRestart("")

	serverCert := getServerCertificate()
	assert.NotEmpty(serverCert)

	// Writer
	{
		serverCert := getServerCertificate()
		sig := getManifestSignature(serverCert)
		assert.Equal(calculateManifestSignature(manifest), sig)

		db := sqlOpen("writer", writerCert, writerKey, serverCert)
		_, err := db.Exec("INSERT INTO test.data VALUES (2), (6)")
		db.Close()
		assert.NoError(err)
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
		assert.Error(err)
		assert.Equal(4., avg)
	}
}

func TestLoggingDebug(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logDir, err := ioutil.TempDir("", "")
	require.NoError(err)
	defer os.RemoveAll(logDir)

	setConfig(true, logDir)
	defer cleanupConfig()
	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()

	_, err = postManifest(serverCert, createManifest("", []string{
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
	}, true, ""), true)
	assert.Nil(err)

	assert.Nil(process.Kill())

	assert.FileExists(filepath.Join(logDir, "mariadb.err"))
	assert.FileExists(filepath.Join(logDir, "mariadb.log"))
	assert.FileExists(filepath.Join(logDir, "mariadb-slow.log"))
}

func TestLoggingNoDebug(t *testing.T) {
	assert := assert.New(t)

	logDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			panic(err)
		}
	}()

	setConfig(false, "")
	defer cleanupConfig()
	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()

	_, err = postManifest(serverCert, createManifest("", []string{
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
	}, true, ""), true)
	assert.Nil(err)

	assert.Nil(process.Kill())

	assert.NoFileExists(filepath.Join(logDir, "mariadb.err"))
	assert.NoFileExists(filepath.Join(logDir, "mariadb.log"))
	assert.NoFileExists(filepath.Join(logDir, "mariadb-slow.log"))
}

func TestLoggingDebugStderr(t *testing.T) {
	assert := assert.New(t)

	logDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			panic(err)
		}
	}()

	setConfig(true, "")
	defer cleanupConfig()

	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()

	_, err = postManifest(serverCert, createManifest("", []string{
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
	}, true, ""), true)
	assert.Nil(err)

	assert.Nil(process.Kill())

	assert.NoFileExists(filepath.Join(logDir, "mariadb.err"))
	assert.NoFileExists(filepath.Join(logDir, "mariadb.log"))
	assert.NoFileExists(filepath.Join(logDir, "mariadb-slow.log"))
}

func TestLoggingNotSetInManifest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logDir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(logDir); err != nil {
			panic(err)
		}
	}()

	setConfig(true, logDir)
	defer cleanupConfig()
	// starting EDB
	cmd := createEdbCmd("")
	require.NoError(cmd.Start())
	defer cmd.Process.Kill()

	log.Println("EDB starting ...")
	waitForEDB(cmd)

	serverCert := getServerCertificate()

	_, err = postManifest(serverCert, createManifest("", []string{
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
	}, false, ""), false)
	assert.NotNil(err)

	assert.Error(cmd.Wait())
}

func TestRecovery(t *testing.T) {
	assert := assert.New(t)

	caCertPem, caKeyPem := createCertificate("ca", "", "")
	usrCertPem, usrKeyPem := createCertificate("usr", caCertPem, caKeyPem)
	recoveryKeyPem, recoveryKeyPriv := createRecoveryKey()

	manifest := createManifest(caCertPem, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
		"GRANT ALL ON test.data TO usr",
	}, false, recoveryKeyPem)

	setConfig(false, "")
	defer cleanupConfig()

	process := startEDB("")
	assert.NotNil(process)

	serverCert := getServerCertificate()

	recoveryKeyEncB64, err := postManifest(serverCert, manifest, true)
	assert.NoError(err)
	recoveryKeyEnc, err := base64.StdEncoding.DecodeString(string(recoveryKeyEncB64))
	assert.NoError(err)
	initialSig := getManifestSignature(serverCert)

	db := sqlOpen("usr", usrCertPem, usrKeyPem, serverCert)
	_, err = db.Exec("INSERT INTO test.data VALUES (2)")
	db.Close()
	assert.NoError(err)

	assert.NoError(process.Kill())

	// Delete master key
	dataPath := os.Getenv(core.EnvDataPath)
	ioutil.WriteFile(filepath.Join(dataPath, "edb-persistence/sealed_key"), []byte{1, 2, 3}, 0o600)

	// edb should start and go into recovery mode
	process = startEDB("")
	assert.NotNil(process)
	defer process.Kill()

	newServerCert := getServerCertificate()
	sig := getManifestSignature(newServerCert)
	assert.Empty(sig)

	// Post recovery key
	recoveryKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, recoveryKeyPriv, recoveryKeyEnc, nil)
	assert.NoError(err)
	postRecoveryKey(newServerCert, recoveryKey)

	// Check that we recovered successfully
	sig = getManifestSignature(serverCert)
	assert.NotEmpty(sig)
	assert.Equal(initialSig, sig)

	var val float64
	db = sqlOpen("usr", usrCertPem, usrKeyPem, serverCert)
	assert.Nil(db.QueryRow("SELECT i FROM test.data").Scan(&val))
	db.Close()
	assert.Equal(2., val)
}

func TestDropDatabase(t *testing.T) {
	assert := assert.New(t)

	caCertPem, caKeyPem := createCertificate("ca", "", "")
	usrCertPem, usrKeyPem := createCertificate("usr", caCertPem, caKeyPem)

	manifest := createManifest(caCertPem, []string{
		"CREATE USER usr REQUIRE ISSUER '/CN=ca' SUBJECT '/CN=usr'",
		"CREATE DATABASE test",
		"CREATE TABLE test.data (i INT)",
		"GRANT ALL ON test.* TO usr",
	}, false, "")

	setConfig(false, "")
	defer cleanupConfig()

	process := startEDB("")
	assert.NotNil(process)
	defer process.Kill()

	serverCert := getServerCertificate()

	_, err := postManifest(serverCert, manifest, true)
	assert.NoError(err)

	db := sqlOpen("usr", usrCertPem, usrKeyPem, serverCert)
	_, err = db.Exec("DROP DATABASE test")
	assert.NoError(err)
	_, err = db.Exec("CREATE DATABASE test")
	assert.NoError(err)
	_, err = db.Exec("CREATE TABLE test.data (i INT)")
	assert.NoError(err)
	// When EDB restarts the memfs is cleared along with any artefacts in the filesystem
	// If DROP DATABASE test is leaving any artifacts we won't notice that after the first iteration
	// Hence try once more to make sure DROP deletes any artifacts
	_, err = db.Exec("DROP DATABASE test")
	assert.NoError(err)
	_, err = db.Exec("CREATE DATABASE test")
	assert.NoError(err)
	_, err = db.Exec("CREATE TABLE test.data (i INT)")
	assert.NoError(err)
	db.Close()
}

func setConfig(debug bool, logDir string) {
	tempPath, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	os.Setenv(core.EnvAPIAddress, addrAPI)
	os.Setenv(core.EnvDatabaseAddress, addrDB)
	os.Setenv(core.EnvDataPath, tempPath)
	if debug {
		os.Setenv(core.EnvDebug, "ON")
	}
	os.Setenv(core.EnvLogDir, logDir)
}

func cleanupConfig() {
	if err := os.Unsetenv(core.EnvAPIAddress); err != nil {
		panic(err)
	}
	if err := os.Unsetenv(core.EnvDatabaseAddress); err != nil {
		panic(err)
	}

	tempPath := os.Getenv(core.EnvDataPath)
	if err := os.Unsetenv(core.EnvDataPath); err != nil {
		panic(err)
	}
	if err := os.RemoveAll(tempPath); err != nil {
		panic(err)
	}
	if err := os.Unsetenv(core.EnvDebug); err != nil {
		panic(err)
	}
	if err := os.Unsetenv(core.EnvLogDir); err != nil {
		panic(err)
	}
}

// Call with empty string for standalone mode, call with path for Marble mode
func startEDB(marbleUUIDPath string) killer {
	cmd := createEdbCmd(marbleUUIDPath)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
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

	log.Println("EDB starting ...")
	waitForEDB(cmd)
	return killer{proc: cmd.Process, wg: wg}
}

func waitForEDB(cmd *exec.Cmd) {
	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "signature"}
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
			break
		}
	}
}

func createEdbCmd(marbleUUIDPath string) *exec.Cmd {
	var cmd *exec.Cmd
	// marbleUUIDPath set implies that edb is being run as a Marble
	if marbleUUIDPath != "" {
		// Setup edb to run as Marble
		cmd = exec.Command(*exe, "-marble")
		cmd.Env = append(os.Environ(),
			"EDG_MARBLE_COORDINATOR_ADDR="+coordinatorAddress,
			"EDG_MARBLE_TYPE=type",
			"EDG_MARBLE_DNS_NAMES=localhost",
			"EDG_MARBLE_UUID_FILE="+filepath.Join(marbleUUIDPath, "uuid"))
	} else {
		// Setup edb to run standalone
		cmd = exec.Command(*exe)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,            // group child with grandchildren so that we can kill 'em all
		Pdeathsig: syscall.SIGKILL, // kill child if test dies
	}
	return cmd
}

func isUnexpectedEDBError(err error) bool {
	return err != nil && err.Error() != "signal: killed"
}

func createRecoveryKey() (string, *rsa.PrivateKey) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	pubPKIX, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		panic(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})
	return string(pemKey), priv
}

func createCertificate(commonName string, signerCert, signerKey string) (cert, key string) {
	return toPem(generateCertificate(commonName, []string{"localhost"}, signerCert, signerKey, false))
}

func createMarbleSecretCertificate(signerCert, signerKey string) (cert, key string) {
	return toPem(generateCertificate("localhost", []string{"localhost"}, signerCert, signerKey, true))
}

func generateCertificate(commonName string, dnsNames []string, signerCert, signerKey string, leafIsCA bool) ([]byte, *ecdsa.PrivateKey) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: commonName},
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     dnsNames,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	if signerCert == "" || leafIsCA {
		template.BasicConstraintsValid = true
		template.IsCA = true
	}

	var certBytes []byte
	if signerCert == "" {
		certBytes, err = x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	} else {
		signer, errKeyPair := tls.X509KeyPair([]byte(signerCert), []byte(signerKey))
		if errKeyPair != nil {
			panic(errKeyPair)
		}
		parsedSignerCert, _ := x509.ParseCertificate(signer.Certificate[0])
		certBytes, err = x509.CreateCertificate(rand.Reader, template, parsedSignerCert, &priv.PublicKey, signer.PrivateKey)
	}

	if err != nil {
		panic(err)
	}

	return certBytes, priv
}

func toPem(certBytes []byte, priv *ecdsa.PrivateKey) (cert, key string) {
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		panic(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return string(pemCert), string(pemKey)
}

func getServerCertificate() string {
	var blocks []*pem.Block
	var err error
	if attestationEnabled && *attestationConfig != "" {
		blocks, _, err = era.GetCertificate(addrAPI, *attestationConfig)
	} else {
		blocks, err = era.InsecureGetCertificate(addrAPI)
	}
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(blocks[0]))
}

func createManifest(ca string, sql []string, debug bool, recovery string) []byte {
	manifest := struct {
		SQL      []string
		CA       string
		Debug    bool
		Recovery string
	}{sql, ca, debug, recovery}
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

func postManifest(serverCert string, manifest []byte, waitForRestart bool) ([]byte, error) {
	client := createHttpClient(serverCert)
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "manifest"}

	log.Print("posting manifest ...")
	resp, err := client.Post(url.String(), "", bytes.NewReader(manifest))
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, errors.New(resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}
	recoveryKey := body

	if !waitForRestart {
		return nil, nil
	}

	// wait until edb restarted
	waitUntilRestart(serverCert)

	return recoveryKey, nil
}

func waitUntilRestart(serverCert string) {
	var client http.Client
	if serverCert != "" {
		client = createHttpClient(serverCert)
	} else {
		client = http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	}

	url := url.URL{Scheme: "https", Host: addrAPI, Path: "signature"}

	// wait until edb restarted
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
				return
			}
		}
	}
}

func postRecoveryKey(serverCert string, key []byte) error {
	client := createHttpClient(serverCert)
	url := url.URL{Scheme: "https", Host: addrAPI, Path: "recover"}

	log.Print("posting recovery key ...")
	resp, err := client.Post(url.String(), "", bytes.NewReader(key))
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return errors.New(resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}
	return nil
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

// Marblerun mock functions down below
func startMockMarblerunCoordinator(jsonManifest []byte) (*grpc.Server, string, error) {
	// Create certificate for the Coordinator
	certBytes, priv := generateCertificate("Mocked Coordinator", []string{"localhost"}, "", "", false)
	cert := tls.Certificate{Certificate: [][]byte{certBytes}, PrivateKey: priv}

	// Create temp directory for data
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, "", err
	}

	// Launch mocked gRPC Marblerun server
	server := grpc.NewServer(grpc.Creds(credentials.NewServerTLSFromCert(&cert)))

	// Generate root certificate & root key for edb
	privKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		panic(err)
	}

	rootCertPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
	privKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privKeyPKCS8}))
	secertRootCert, secretRootKey := createMarbleSecretCertificate(rootCertPEM, privKeyPEM)

	marbleServer := marbleServer{dataDir: tempDir, rootCert: rootCertPEM, secretRootCert: secertRootCert, secretRootKey: secretRootKey, manifest: jsonManifest}
	rpc.RegisterMarbleServer(server, marbleServer)

	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		return nil, "", err
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			panic(err)
		}
	}()

	coordinatorAddress = listener.Addr().String()

	return server, tempDir, nil
}

type marbleServer struct {
	rpc.UnimplementedMarbleServer
	dataDir        string
	rootCert       string
	secretRootCert string
	secretRootKey  string
	manifest       []byte
}

func (m marbleServer) Activate(context.Context, *rpc.ActivationReq) (*rpc.ActivationResp, error) {
	return &rpc.ActivationResp{
		Parameters: &rpc.Parameters{
			Env: map[string][]byte{
				core.EnvAPIAddress:             []byte(addrAPI),
				core.EnvDatabaseAddress:        []byte(addrDB),
				core.EnvDataPath:               []byte(m.dataDir),
				core.ERocksDBMasterKeyVar:      []byte("4142434445464748494a4b4c4d4e4f50"),
				marble.MarbleEnvironmentRootCA: []byte(m.rootCert),
				db.EnvRootCertificate:          []byte(m.secretRootCert),
				db.EnvRootKey:                  []byte(m.secretRootKey),
				core.EnvManifestFile:           []byte("/tmp/manifest.json"),
			},
			Files: map[string][]byte{"/tmp/manifest.json": m.manifest},
		},
	}, nil
}

type killer struct {
	proc *os.Process
	wg   *sync.WaitGroup
}

func (k killer) Kill() error {
	err := k.proc.Kill()
	k.wg.Wait()
	return err
}

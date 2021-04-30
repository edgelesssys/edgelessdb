package db

//go:generate sh -c "./mariadb_gen_bootstrap.sh ../../server > mariadbbootstrap.go"

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql" // import driver used via the database/sql package
)

const edbInternalAddr = "EDB_INTERNAL_ADDR" // must be kept sync with src/mysqld_edb.cc

const (
	filenameCA   = "ca.pem"
	filenameCert = "cert.pem"
	filenameKey  = "key.pem"
	filenameCnf  = "my.cnf"
	filenameInit = "init.sql"
)

// Mariadbd is used to control mariadbd.
type Mariadbd interface {
	Main(cnfPath string) int
	WaitUntilStarted()
	WaitUntilListenInternalReady()
}

// Mariadb is a secure database based on MariaDB.
type Mariadb struct {
	internalPath, externalPath       string
	internalAddress, externalAddress string
	certificateCommonName            string
	mariadbd                         Mariadbd
	log                              *log.Logger
	cert                             []byte
	key                              crypto.PrivateKey
	manifestSig                      []byte
	ca                               string
}

// NewMariadb creates a new Mariadb object.
func NewMariadb(internalPath, externalPath, internalAddress, externalAddress, certificateCommonName string, mariadbd Mariadbd) (*Mariadb, error) {
	return &Mariadb{
		internalPath:          internalPath,
		externalPath:          externalPath,
		internalAddress:       internalAddress,
		externalAddress:       externalAddress,
		certificateCommonName: certificateCommonName,
		mariadbd:              mariadbd,
		log:                   log.New(os.Stdout, "[EDB] ", log.LstdFlags),
	}, nil
}

// GetCertificate gets the database certificate.
func (d *Mariadb) GetCertificate() ([]byte, crypto.PrivateKey) {
	return d.cert, d.key
}

// Initialize sets up a database according to the jsonManifest.
func (d *Mariadb) Initialize(jsonManifest []byte) error {
	if d.manifestSig != nil {
		return errors.New("already initialized")
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return err
	}

	if err := d.configureBootstrap(man.SQL, jsonManifest); err != nil {
		return err
	}

	d.log.Println("initializing ...")
	if d.mariadbd.Main(filepath.Join(d.internalPath, filenameCnf)) != 0 {
		// unrecoverable
		// TODO AB#882 pass concrete error to owner (might be an error in man.SQL)
		// Note that there can be SQL errors even if Main returns 0
		panic("bootstrap failed")
	}

	return nil
}

// Start starts the database.
func (d *Mariadb) Start() error {
	_, err := os.Stat(filepath.Join(d.externalPath, "#rocksdb"))
	if os.IsNotExist(err) {
		d.cert, d.key = createCertificate(d.certificateCommonName)
		d.log.Println("DB has not been initialized, waiting for manifest.")
		return nil
	}
	if err != nil {
		return err
	}

	if err := d.configureStart(); err != nil {
		return err
	}

	// Set internal addr env var so that mariadb will first listen on that addr. SSL and ACL will not be active at this point,
	// so we can get the cert and key from the db, write it to the memfs, and then let mariadb complete its startup sequence.
	normalizedInternalAddr := net.JoinHostPort(splitHostPort(d.internalAddress, "3305"))
	if err := os.Setenv(edbInternalAddr, normalizedInternalAddr); err != nil {
		return err
	}

	d.log.Println("starting up ...")
	go func() {
		ret := d.mariadbd.Main(filepath.Join(d.internalPath, filenameCnf))
		panic(fmt.Errorf("mariadbd.Main returned unexpectedly with %v", ret))
	}()
	d.mariadbd.WaitUntilListenInternalReady()

	// errors are unrecoverable from here

	cert, key, jsonManifest, err := getConfigFromSQL(normalizedInternalAddr)
	if err != nil {
		d.log.Println("An intialization attempt failed. The DB is in an inconsistent state. Please provide an empty data directory")
		d.log.Fatalln(err)
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		panic(err)
	}

	d.setManifestSignature(jsonManifest)
	d.ca = man.CA
	d.cert = cert
	d.key = key

	if err := d.writeCertificates(); err != nil {
		panic(err)
	}

	// clear env var and connect once more to signal mariadb that we are ready to start
	if err := os.Setenv(edbInternalAddr, ""); err != nil {
		panic(err)
	}
	c, err := net.Dial("tcp", normalizedInternalAddr)
	if err != nil {
		panic(err)
	}
	c.Close()

	d.mariadbd.WaitUntilStarted()
	d.log.Println("DB is running.")
	return nil
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (d *Mariadb) GetManifestSignature() []byte {
	return d.manifestSig
}

func (d *Mariadb) setManifestSignature(jsonManifest []byte) {
	sig := sha256.Sum256(jsonManifest)
	d.manifestSig = sig[:]
}

// configure MariaDB for bootstrap
func (d *Mariadb) configureBootstrap(sql []string, jsonManifest []byte) error {
	var queries string
	if len(sql) > 0 {
		queries = strings.Join(sql, ";\n") + ";"
	}

	key, err := x509.MarshalPKCS8PrivateKey(d.key)
	if err != nil {
		return err
	}

	init := fmt.Sprintf(`
CREATE DATABASE mysql;
USE mysql;
%v
FLUSH PRIVILEGES;
%v
CREATE DATABASE $edgeless;
CREATE TABLE $edgeless.config (c BLOB, k BLOB, m BLOB);
INSERT INTO $edgeless.config VALUES (%#x, %#x, %#x);
`, mariadbBootstrap, queries, d.cert, key, jsonManifest)

	cnf := `
[mysqld]
datadir=` + d.externalPath + `
default-storage-engine=ROCKSDB
enforce-storage-engine=ROCKSDB
bootstrap
init-file=` + filepath.Join(d.internalPath, filenameInit) + `
`

	if err := d.writeFile(filenameCnf, []byte(cnf)); err != nil {
		return err
	}
	if err := d.writeFile(filenameInit, []byte(init)); err != nil {
		return err
	}
	return nil
}

// configure MariaDB for regular start
func (d *Mariadb) configureStart() error {
	host, port := splitHostPort(d.externalAddress, "3306")

	cnf := `
[mysqld]
datadir=` + d.externalPath + `
default-storage-engine=ROCKSDB
enforce-storage-engine=ROCKSDB
socket=
bind-address=` + host + `
port=` + port + `
require-secure-transport=1
ssl-ca = "` + filepath.Join(d.internalPath, filenameCA) + `"
ssl-cert = "` + filepath.Join(d.internalPath, filenameCert) + `"
ssl-key = "` + filepath.Join(d.internalPath, filenameKey) + `"
`

	return d.writeFile(filenameCnf, []byte(cnf))
}

func (d *Mariadb) writeCertificates() error {
	cert, key, err := toPEM(d.cert, d.key)
	if err != nil {
		return err
	}
	if err := d.writeFile(filenameCA, []byte(d.ca)); err != nil {
		return err
	}
	if err := d.writeFile(filenameCert, cert); err != nil {
		return err
	}
	if err := d.writeFile(filenameKey, key); err != nil {
		return err
	}
	return nil
}

func (d *Mariadb) writeFile(filename string, data []byte) error {
	return ioutil.WriteFile(filepath.Join(d.internalPath, filename), data, 0600)
}

func getConfigFromSQL(address string) (cert []byte, key crypto.PrivateKey, config []byte, err error) {
	db, err := sqlOpen(address)
	defer db.Close()
	if err != nil {
		return
	}

	var keyRaw []byte
	if err := db.QueryRow("SELECT * from $edgeless.config").Scan(&cert, &keyRaw, &config); err != nil {
		return nil, nil, nil, err
	}

	if key, err = x509.ParsePKCS8PrivateKey(keyRaw); err != nil {
		return nil, nil, nil, err
	}

	return
}

func sqlOpen(address string) (*sql.DB, error) {
	return sql.Open("mysql", "root@tcp("+address+")/")
}

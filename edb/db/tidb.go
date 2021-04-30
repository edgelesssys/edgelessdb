package db

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/go-sql-driver/mysql"
)

const (
	filenameCA      = "ca.pem"
	filenameCert    = "cert.pem"
	filenameKey     = "key.pem"
	filenameTidbcfg = "tidbcfg"
)

// TidbLauncher is used to start and stop TiDB.
type TidbLauncher interface {
	Start()
	Stop()
}

// Tidb is a secure database based on TiDB.
type Tidb struct {
	internalPath, externalPath       string
	internalAddress, externalAddress string
	launcher                         TidbLauncher
	log                              *log.Logger
	cert                             []byte
	key                              crypto.PrivateKey
	manifestSig                      []byte
	ca                               string
}

// NewTidb creates a new Tidb object.
func NewTidb(internalPath, externalPath, internalAddress, externalAddress, certificateCommonName string, launcher TidbLauncher) (*Tidb, error) {
	d := &Tidb{
		internalPath:    internalPath,
		externalPath:    externalPath,
		internalAddress: internalAddress,
		externalAddress: externalAddress,
		launcher:        launcher,
		log:             log.New(os.Stdout, "[EDB] ", log.LstdFlags),
	}

	// Start TiDB using internal sockets.
	if err := d.configureInternal(); err != nil {
		return nil, err
	}
	d.launcher.Start()
	defer d.launcher.Stop()

	cert, key, jsonManifest, err := getConfigFromSQL(d.internalAddress)
	if err != nil {
		return nil, err
	}

	if cert == nil {
		d.cert, d.key = createCertificate(certificateCommonName)
		key, err := x509.MarshalPKCS8PrivateKey(d.key)
		if err != nil {
			return nil, err
		}
		if err := putCertToSQL(d.internalAddress, d.cert, key); err != nil {
			return nil, err
		}
		return d, nil
	}

	d.cert = cert
	d.key, err = x509.ParsePKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}

	if jsonManifest == nil {
		// db has not been initialized yet
		return d, nil
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return nil, err
	}

	d.setManifestSignature(jsonManifest)
	d.ca = man.CA

	return d, nil
}

// GetCertificate gets the database certificate.
func (d *Tidb) GetCertificate() ([]byte, crypto.PrivateKey) {
	return d.cert, d.key
}

// Initialize sets up a database according to the jsonManifest.
func (d *Tidb) Initialize(jsonManifest []byte) error {
	if d.manifestSig != nil {
		return errors.New("already initialized")
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return err
	}

	if err := d.configureInternal(); err != nil {
		return err
	}

	// For initial configuration, we start TiDB using internal sockets.
	d.log.Println("initializing ...")
	d.launcher.Start()
	err := execInitialSQL(man.SQL, d.internalAddress, jsonManifest)
	d.launcher.Stop()
	if err != nil {
		d.log.Println(err)
		return err
	}
	d.log.Println("DB is initialized.")

	d.setManifestSignature(jsonManifest)
	d.ca = man.CA
	return nil
}

// Start starts the database.
func (d *Tidb) Start() error {
	if d.manifestSig == nil {
		d.log.Println("DB has not been initialized, waiting for manifest.")
		return nil
	}

	// DB has been initialized, start it using external sockets.
	if err := d.configureExternal(); err != nil {
		return err
	}
	d.log.Println("starting up ...")
	d.launcher.Start()
	d.log.Println("DB is running.")

	return nil
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (d *Tidb) GetManifestSignature() []byte {
	return d.manifestSig
}

func (d *Tidb) setManifestSignature(jsonManifest []byte) {
	sig := sha256.Sum256(jsonManifest)
	d.manifestSig = sig[:]
}

// configure TiDB for internal socket without security
func (d *Tidb) configureInternal() error {
	host, port := splitHostPort(d.internalAddress, "3306")

	cfg := `
path = "` + d.externalPath + `"
host = "` + host + `"
port = ` + port + `
oom-use-tmp-storage = false

[security]
skip-grant-table = true

[log]
level = "fatal"
enable-slow-log = false

[status]
report-status = false
`

	return ioutil.WriteFile(filepath.Join(d.internalPath, filenameTidbcfg), []byte(cfg), 0600)
}

// configure TiDB for external socket with security
func (d *Tidb) configureExternal() error {
	pathCA := filepath.Join(d.internalPath, filenameCA)
	pathCert := filepath.Join(d.internalPath, filenameCert)
	pathKey := filepath.Join(d.internalPath, filenameKey)

	host, port := splitHostPort(d.externalAddress, "3306")

	cfg := `
path = "` + d.externalPath + `"
host = "` + host + `"
port = ` + port + `
oom-use-tmp-storage = false

[security]
ssl-ca = "` + pathCA + `"
ssl-cert = "` + pathCert + `"
ssl-key = "` + pathKey + `"
require-secure-transport = true

[log]
level = "fatal"
enable-slow-log = false

[status]
report-status = false
`

	pemCert, pemKey, err := toPEM(d.cert, d.key)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(pathCA, []byte(d.ca), 0600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(pathCert, pemCert, 0600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(pathKey, pemKey, 0600); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(d.internalPath, filenameTidbcfg), []byte(cfg), 0600)
}

func execInitialSQL(queries []string, address string, config []byte) error {
	pw, err := generatePassword()
	if err != nil {
		return err
	}

	db, err := sqlOpen(address)
	defer db.Close()
	if err != nil {
		return err
	}

	// DDL queries like CREATE TABLE implicitly commit a
	// transaction so it makes no sense to use one here.

	// Restrict root user to the internal address and set a random password. We don't need to
	// save the password and will use skip-grant-table instead when we make internal connections.
	host, _ := splitHostPort(address, "")
	_, err = db.Exec("UPDATE mysql.user SET Host=?, authentication_string=PASSWORD(?) WHERE user='root'", host, pw)
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE $edgeless.config (c BLOB)")
	if err != nil {
		if sqlerr, ok := err.(*mysql.MySQLError); ok && sqlerr.Number == 1050 { // ER_TABLE_EXISTS_ERROR
			return errors.New("A previous intialization attempt failed. Please contact the DB administrator to reset the DB")
		}
		return err
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}

	// Insert config last so that we know that initialization has been successful if config exists.
	_, err = db.Exec("INSERT INTO $edgeless.config VALUES (?)", config)
	return err
}

func getConfigFromSQL(address string) (cert, key, config []byte, err error) {
	db, err := sqlOpen(address)
	defer db.Close()
	if err != nil {
		return
	}

	err = db.QueryRow("SELECT * from $edgeless.cert").Scan(&cert, &key)
	if err != nil && !sqlIsNoSuchTable(err) {
		return
	}

	err = db.QueryRow("SELECT c from $edgeless.config").Scan(&config)
	if err == sql.ErrNoRows {
		err = errors.New("An intialization attempt failed. The DB is in an inconsistent state. Please provide an empty data directory")
	} else if err != nil && sqlIsNoSuchTable(err) {
		err = nil
	}

	return
}

func putCertToSQL(address string, cert, key []byte) error {
	db, err := sqlOpen(address)
	defer db.Close()
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE DATABASE $edgeless")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE $edgeless.cert (c BLOB, k BLOB)")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO $edgeless.cert VALUES (?, ?)", cert, key)
	if err != nil {
		return err
	}

	return nil
}

func sqlOpen(address string) (*sql.DB, error) {
	return sql.Open("mysql", "root@tcp("+address+")/")
}

func sqlIsNoSuchTable(err error) bool {
	sqlerr, ok := err.(*mysql.MySQLError)
	return ok && sqlerr.Number == 1146
}

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

package db

//go:generate sh -c "./mariadb_gen_bootstrap.sh ../../3rdparty/edgeless-mariadb > mariadbbootstrap.go"

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/edgelesssys/edgelessdb/edb/rt"
	_ "github.com/go-sql-driver/mysql" // import driver used via the database/sql package
)

const edbInternalAddr = "EDB_INTERNAL_ADDR" // must be kept sync with src/mysqld_edb.cc

const (
	filenameCA           = "ca.pem"
	filenameCert         = "cert.pem"
	filenameKey          = "key.pem"
	filenameCnf          = "my.cnf"
	filenameInit         = "init.sql"
	filenameBootstrapLog = "mariadb-error.log"
	filenameGeneralLog   = "mariadb.log"
	filenameSlowQueryLog = "mariadb-slow.log"
	filenameBinaryLog    = "mariadb-binary.log"
	FilenameErrorLog     = "mariadb.err" // this one is public as we try to parse it from elsewhere in case MariaDB directly tries to call exit() due to an error
)

// ErrPreviousInitFailed is thrown when a previous initialization attempt failed, but another init or start is attempted.
var ErrPreviousInitFailed = errors.New("a previous initialization attempt failed")

// ErrNotInitializedYet is thrown when the database has not been initialized yet
var ErrNotInitializedYet = errors.New("database has not been initialized yet")

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
	debug                            bool
	debugLogDir                      string
	mariadbd                         Mariadbd
	cert                             []byte
	key                              crypto.PrivateKey
	manifestSig                      []byte
	ca                               string
	attemptedInit                    bool
}

// NewMariadb creates a new Mariadb object.
func NewMariadb(internalPath, externalPath, internalAddress, externalAddress, certificateDNSName, logDir string, debug bool, isMarble bool, mariadbd Mariadbd) (*Mariadb, error) {
	if err := os.MkdirAll(externalPath, 0700); err != nil {
		return nil, err
	}
	d := &Mariadb{
		internalPath:    internalPath,
		externalPath:    externalPath,
		internalAddress: internalAddress,
		externalAddress: externalAddress,
		debug:           debug,
		debugLogDir:     logDir,
		mariadbd:        mariadbd,
	}

	var cert []byte
	var key crypto.PrivateKey
	var err error
	if isMarble {
		// When running under Marblerun, expect that it passes edb's root certificate + private key
		rt.Log.Println("parsing root certificate passed from Marblerun")
		cert, key, err = setupCertificateFromMarblerun()
	} else {
		// Otherweise in standalone mode, we generate this here
		cert, key, err = createCertificate(certificateDNSName)
	}
	if err != nil {
		return nil, err
	}

	d.cert = cert
	d.key = key
	return d, nil
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
	if d.attemptedInit {
		rt.Log.Println("Cannot initialize the database, a previous attempt failed. The DB is in an inconsistent state. Please provide an empty data directory.")
		return ErrPreviousInitFailed
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return err
	}

	if d.debug && !man.Debug {
		return fmt.Errorf("edb was started in debug mode but the manifest does not allow debug mode")
	}

	if err := d.configureBootstrap(man.SQL, jsonManifest); err != nil {
		return err
	}

	rt.Log.Println("initializing ...")

	// Remove already existing log file, as we do not want replayed logs
	err := os.Remove(filepath.Join(d.internalPath, filenameBootstrapLog))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	d.attemptedInit = true

	// Launch MariaDB
	if err := d.mariadbd.Main(filepath.Join(d.internalPath, filenameCnf)); err != 0 {
		d.printErrorLog(false)
		rt.Log.Printf("FATAL: bootstrap failed, MariaDB exited with error code: %d\n", err)
		panic("bootstrap failed")
	}

	return d.printErrorLog(true)
}

// Start starts the database.
func (d *Mariadb) Start() error {
	_, err := os.Stat(filepath.Join(d.externalPath, "#rocksdb"))
	if os.IsNotExist(err) {
		rt.Log.Println("DB has not been initialized, waiting for manifest.")
		return ErrNotInitializedYet
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

	rt.Log.Println("starting up ...")
	go func() {
		ret := d.mariadbd.Main(filepath.Join(d.internalPath, filenameCnf))
		panic(fmt.Errorf("mariadbd.Main returned unexpectedly with %v", ret))
	}()
	d.mariadbd.WaitUntilListenInternalReady()

	// errors are unrecoverable from here

	cert, key, jsonManifest, err := getConfigFromSQL(normalizedInternalAddr)
	if err != nil {
		rt.Log.Println("An initialization attempt failed. The DB is in an inconsistent state. Please provide an empty data directory.")
		rt.Log.Fatalln(err)
	}

	var man manifest
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		panic(err)
	}

	if d.debug && !man.Debug {
		panic(fmt.Errorf("edb was started in debug mode but the manifest does not allow debug mode"))
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
	rt.Log.Println("DB is running.")
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
log-error =` + filepath.Join(d.internalPath, filenameBootstrapLog) + `
bootstrap
init-file=` + filepath.Join(d.internalPath, filenameInit) + `
`
	if len(d.debugLogDir) > 0 {
		cnf += fmt.Sprintf("%v=%v\n", "rocksdb_db_log_dir", d.debugLogDir)
	} else {
		cnf += fmt.Sprintf("%v=%v\n", "rocksdb_db_log_dir", d.internalPath)
	}

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
user=root
bind-address=` + host + `
port=` + port + `
skip-name-resolve
require-secure-transport=1
ssl-ca = "` + filepath.Join(d.internalPath, filenameCA) + `"
ssl-cert = "` + filepath.Join(d.internalPath, filenameCert) + `"
ssl-key = "` + filepath.Join(d.internalPath, filenameKey) + `"
`
	if d.debug {
		// If nothing is specified ONLY error-log is printed on stderr
		// Setting any of the logs without a file logs them to default files
		// log-basename only works for logging to `datadir`
		// https://mariadb.com/kb/en/error-log/
		// https://mariadb.com/kb/en/error-log/#writing-the-error-log-to-stderr-on-unix
		// https://mariadb.com/kb/en/general-query-log/
		// https://mariadb.com/kb/en/slow-query-log-overview/
		// http://myrocks.io/docs/getting-started/
		if len(d.debugLogDir) > 0 {
			logFiles := map[string]string{"log_error": FilenameErrorLog, "general_log_file": filenameGeneralLog, "slow_query_log_file": filenameSlowQueryLog, "log_bin": filenameBinaryLog, "rocksdb_db_log_dir": ""}
			for logName, logFile := range logFiles {
				cnf += fmt.Sprintf("%v=%v\n", logName, filepath.Join(d.debugLogDir, logFile))
			}
		}

		cnf += `
general_log
slow_query_log
log_warnings=9
binlog-format=ROW
`
	} else {
		// Redirect error-log to memfs
		cnf += fmt.Sprintf("%v=%v\n", "log_error", filepath.Join(d.internalPath, FilenameErrorLog))
		cnf += fmt.Sprintf("%v=%v\n", "rocksdb_db_log_dir", d.internalPath)
	}
	return d.writeFile(filenameCnf, []byte(cnf))
}

func (d *Mariadb) writeCertificates() error {
	cert, key, err := toPEM(d.cert, d.key)
	if err != nil {
		return err
	}

	var ca []byte
	if d.ca != "" {
		ca = []byte(d.ca)
	} else {
		// The manifest didn't contain a CA certificate, but we set ssl-ca in configureStart.
		// Thus, we must provide one or otherwise mariadb will not accept connections.
		ca, _, err = createCertificate("dummy")
		if err != nil {
			return err
		}
		ca = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca})
	}

	if err := d.writeFile(filenameCA, ca); err != nil {
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
	if err != nil {
		return
	}
	defer db.Close()

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

func (d *Mariadb) printErrorLog(onlyPrintOnError bool) error {
	// Restore original stdout & stderr from MariaDB's redirection
	if err := rt.RestoreStdoutAndStderr(); err != nil {
		panic(err)
	}

	// Read error log from internal memfs
	// This file should always be created when everything is somewhat running okay
	// Even when silent startup is set and nothing was printed to the error log
	errorLogBytes, err := ioutil.ReadFile(filepath.Join(d.internalPath, filenameBootstrapLog))
	if err != nil {
		panic("cannot read MariaDB's error log: " + err.Error())
	}
	errorLog := string(errorLogBytes)

	// Check if "ERROR" (case insensitive) occurs in MariaDB's error log
	pattern := regexp.MustCompile(`(?mi)^ERROR.*$`)
	foundErrors := pattern.FindAllString(errorLog, -1)

	// Print error log if an error was found or we explicitly asked for the log
	if foundErrors != nil || !onlyPrintOnError {
		fmt.Print(errorLog)
	}

	// And if we found errors, return them to the caller
	if foundErrors != nil {
		return errors.New(strings.Join(foundErrors, ""))
	}

	return nil
}

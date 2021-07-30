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

package core

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/edgelesssys/edgelessdb/edb/db"
	"github.com/edgelesssys/edgelessdb/edb/rt"
	"github.com/edgelesssys/edgelessdb/edb/util"
	"github.com/edgelesssys/ego/marble"
	"github.com/spf13/afero"
)

// Core implements the core logic of EDB.
type Core struct {
	state     state
	cfg       Config
	rt        rt.Runtime
	db        db.Database
	fs        afero.Afero
	mutex     sync.Mutex
	report    []byte
	isMarble  bool
	masterKey []byte
}

// The sequence of states EDB may be in
type state int

const (
	stateUninitialized state = iota
	stateRecovery
	stateInitialized
	stateMax
)

// Needs to be paired with `defer c.mux.Unlock()`
func (c *Core) requireState(states ...state) error {
	c.mutex.Lock()
	for _, s := range states {
		if s == c.state {
			return nil
		}
	}
	return errors.New("edb is not in expected state")
}

func (c *Core) advanceState(newState state) {
	if !(c.state < newState && newState < stateMax) {
		panic(fmt.Errorf("cannot advance from %d to %d", c.state, newState))
	}
	c.state = newState
}

// NewCore creates a new Core object.
func NewCore(cfg Config, rt rt.Runtime, db db.Database, fs afero.Afero, isMarble bool) *Core {
	c := &Core{state: stateUninitialized, cfg: cfg, rt: rt, fs: fs, db: db, isMarble: isMarble}
	c.mustInitMasterKey()
	return c
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (c *Core) GetManifestSignature() []byte {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.db.GetManifestSignature()
}

// GetCertificateReport gets the certificate and a report that includes the certificate's hash.
func (c *Core) GetCertificateReport() (string, []byte, error) {
	// When running as a Marble, return certificate chain
	if c.isMarble {
		return c.getCertificateReportMarble()
	}

	// When running standalone, return edb root certificate
	cert, _ := c.db.GetCertificate()
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if len(pemCert) <= 0 {
		return "", nil, errors.New("failed to encode certificate")
	}
	return string(pemCert), c.report, nil
}

// GetTLSConfig creates a TLS configuration that includes the certificate.
func (c *Core) GetTLSConfig() *tls.Config {
	// When running as a Marble, return TLS config containing a certificate chain
	if c.isMarble {
		return c.getTLSConfigMarble()
	}

	// When running standalone, return TLS config containing only edb's self-signed certificate
	cert, key := c.db.GetCertificate()
	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  key,
			},
		},
		GetConfigForClient: c.getConfigForClient,
	}
}

// Initialize sets up a database according to the jsonManifest.
func (c *Core) Initialize(jsonManifest []byte) ([]byte, error) {
	// Encrypt recovery key if certificate is provided.
	var man struct{ Recovery string }
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return nil, err
	}
	recoveryKey, err := c.encryptRecoveryKey(c.masterKey, man.Recovery)
	if err != nil {
		return nil, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.db.Initialize(jsonManifest); err != nil {
		return nil, err
	}

	fmt.Println("restarting ...")
	go func() {
		time.Sleep(time.Second)
		c.rt.RestartHostProcess()
	}()
	return recoveryKey, nil
}

// IsRecovering returns if edb (in standalone mode) is in recovery mode, or if it's not.
func (c *Core) IsRecovering() bool {
	defer c.mutex.Unlock()
	return c.requireState(stateRecovery) == nil
}

// Recover sets an encryption key (ideally decrypted from the recovery data) and tries to unseal and load a saved state again.
func (c *Core) Recover(ctx context.Context, key []byte) error {
	defer c.mutex.Unlock()
	if err := c.requireState(stateRecovery); err != nil {
		return err
	}
	if err := c.setMasterKey(key); err != nil {
		return err
	}
	if err := c.StartDatabase(); err != nil {
		return err
	}
	return nil
}

// StartDatabase starts the database.
func (c *Core) StartDatabase() error {
	// Start MariaDB
	if err := c.db.Start(); err != nil {
		return err
	}

	var cert []byte
	if c.isMarble {
		var err error
		cert, err = c.getCertificateCA()
		if err != nil {
			return err
		}
	} else {
		cert, _ = c.db.GetCertificate()
	}
	hash := sha256.Sum256(cert)
	var err error
	c.report, err = c.rt.GetRemoteReport(hash[:])
	if err != nil {
		fmt.Printf("Failed to get quote: %v\n", err)
	}
	return nil
}

func (c *Core) getConfigForClient(chi *tls.ClientHelloInfo) (*tls.Config, error) {
	if chi.ServerName == "root" {
		// edbra uses this name to get the root certificate
		return nil, nil
	}

	// TLS requires that the hostname matches the server certificate's common name or SAN. However,
	// we don't want to bind the database to a specific hostname or IP and it's not needed for
	// security because we have remote attestation. Instead of requiring the client to be
	// configured to skip hostname verification, we generate a unique certificate for this
	// connection that contains the client's expected server name and IP.

	signerCert, signerKey := c.db.GetCertificate()

	hostname := chi.ServerName
	if hostname == "" {
		// use CommonName of root certificate if client did not send ServerName
		if c, err := x509.ParseCertificate(signerCert); err != nil {
			hostname = c.Subject.CommonName
		} else {
			// can't happen
			hostname = "localhost"
		}
	}

	var ips []net.IP
	if addr, ok := chi.Conn.LocalAddr().(*net.TCPAddr); ok {
		ips = []net.IP{addr.IP}
	}

	cert, key, err := createCertificate(hostname, ips, signerCert, signerKey)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert, signerCert},
				PrivateKey:  key,
			},
		},
	}, nil
}

func (c *Core) encryptRecoveryKey(key []byte, recoveryKeyPEM string) ([]byte, error) {
	if len(recoveryKeyPEM) <= 0 {
		return nil, nil
	}

	block, _ := pem.Decode([]byte(recoveryKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode recovery public key")
	}

	parsedRSAKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := parsedRSAKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to get RSA key from recovery public key")
	}

	recoveryKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, key, nil)
	if err != nil {
		return nil, err
	}

	return recoveryKey, nil
}

func createCertificate(hostname string, ips []net.IP, signerCert []byte, signerKey crypto.PrivateKey) ([]byte, crypto.PrivateKey, error) {
	serialNumber, err := util.GenerateCertificateSerialNumber()
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: hostname},
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{hostname},
		IPAddresses:  ips,
	}
	parsedSignerCert, err := x509.ParseCertificate(signerCert)
	if err != nil {
		return nil, nil, err
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, parsedSignerCert, &priv.PublicKey, signerKey)
	if err != nil {
		return nil, nil, err
	}
	return cert, priv, nil
}

func (c *Core) getCertificateReportMarble() (string, []byte, error) {
	cert, _ := c.db.GetCertificate()
	marbleCACert, err := c.getCertificateCA()
	if err != nil {
		return "", nil, err
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if len(pemCert) <= 0 {
		return "", nil, errors.New("failed to encode certificate")
	}
	marbleCACertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: marbleCACert})
	if len(marbleCACertPEM) <= 0 {
		return "", nil, errors.New("failed to encode certificate")
	}
	return string(pemCert) + string(marbleCACertPEM), c.report, nil
}

func (c *Core) getTLSConfigMarble() *tls.Config {
	cert, key := c.db.GetCertificate()

	marbleCACert, err := c.getCertificateCA()
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert, marbleCACert},
				PrivateKey:  key,
			},
		},
		GetConfigForClient: c.getConfigForClient,
	}
}

// getCertificateCA returns the Marblerun root/intermediate CA certificate used to generate edb's root certificate when running as a Marble
func (c *Core) getCertificateCA() ([]byte, error) {
	if !c.isMarble {
		return nil, errors.New("not running as a Marble")
	}

	marbleCAPEM := os.Getenv(marble.MarbleEnvironmentRootCA)
	if marbleCAPEM == "" {
		return nil, errors.New("cannot successfully retrieve Marble root CA from environment")
	}

	block, _ := pem.Decode([]byte(marbleCAPEM))
	if block == nil {
		return nil, errors.New("cannot decode Marble root CA from environment")
	}

	return block.Bytes, nil
}

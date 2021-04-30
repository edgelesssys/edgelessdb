package core

import (
	"crypto"
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
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
)

// Core implements the core logic of EDB.
type Core struct {
	rt     rt.Runtime
	db     db.Database
	mutex  sync.Mutex
	report []byte
}

// NewCore creates a new Core object.
func NewCore(rt rt.Runtime, db db.Database) *Core {
	return &Core{rt: rt, db: db}
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (c *Core) GetManifestSignature() []byte {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.db.GetManifestSignature()
}

// GetReport gets a report that includes the certificate's hash.
func (c *Core) GetReport() []byte {
	return c.report
}

// GetTLSConfig creates a TLS configuration that includes the certificate.
func (c *Core) GetTLSConfig() *tls.Config {
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
	recoveryKey, err := c.encryptRecoveryKey(man.Recovery)
	if err != nil {
		return nil, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.db.Initialize(jsonManifest); err != nil {
		return nil, err
	}

	if err := c.db.Start(); err != nil {
		return nil, err
	}

	return recoveryKey, nil
}

// StartDatabase starts the database.
func (c *Core) StartDatabase() error {
	if err := c.db.Start(); err != nil {
		return err
	}

	cert, _ := c.db.GetCertificate()
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

	cert, key := createCertificate(hostname, ips, signerCert, signerKey)

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  key,
			},
		},
	}, nil
}

func (c *Core) encryptRecoveryKey(cert string) ([]byte, error) {
	if len(cert) <= 0 {
		return nil, nil
	}

	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		return nil, errors.New("failed to decode recovery certificate")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := parsedCert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to get RSA key from recovery certificate")
	}

	sealKey, err := c.rt.GetProductSealKey()
	if err != nil {
		return nil, err
	}

	recoveryKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, sealKey, nil)
	if err != nil {
		return nil, err
	}

	return recoveryKey, nil
}

func createCertificate(hostname string, ips []net.IP, signerCert []byte, signerKey crypto.PrivateKey) ([]byte, crypto.PrivateKey) {
	// TODO meaningful values
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: hostname},
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{hostname},
		IPAddresses:  ips,
	}
	parsedSignerCert, _ := x509.ParseCertificate(signerCert)
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, parsedSignerCert, &priv.PublicKey, signerKey)
	return cert, priv
}

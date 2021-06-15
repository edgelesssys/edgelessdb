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
	cfg      Config
	rt       rt.Runtime
	db       db.Database
	mutex    sync.Mutex
	report   []byte
	isMarble bool
}

// NewCore creates a new Core object.
func NewCore(cfg Config, rt rt.Runtime, db db.Database, isMarble bool) *Core {
	return &Core{cfg: cfg, rt: rt, db: db, isMarble: isMarble}
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (c *Core) GetManifestSignature() []byte {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.db.GetManifestSignature()
}

// GetCertificateReport gets the certificate and a report that includes the certificate's hash.
func (c *Core) GetCertificateReport() (string, []byte, error) {
	cert, _ := c.db.GetCertificate()
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if len(pemCert) <= 0 {
		return "", nil, errors.New("failed to encode certificate")
	}
	return string(pemCert), c.report, nil
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
	// Initialize master key
	masterKey, err := c.initMasterKey()
	if err != nil {
		return nil, err
	}

	// Encrypt recovery key if certificate is provided.
	var man struct{ Recovery string }
	if err := json.Unmarshal(jsonManifest, &man); err != nil {
		return nil, err
	}
	recoveryKey, err := c.encryptRecoveryKey(masterKey, man.Recovery)
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

// StartDatabase starts the database.
func (c *Core) StartDatabase() error {
	// Initialize master key
	_, err := c.initMasterKey()
	if err != nil {
		return err
	}

	// Start MariaDB
	if err := c.db.Start(); err != nil {
		return err
	}

	cert, _ := c.db.GetCertificate()
	hash := sha256.Sum256(cert)
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

func (c *Core) encryptRecoveryKey(key []byte, cert string) ([]byte, error) {
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

	recoveryKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, key, nil)
	if err != nil {
		return nil, err
	}

	return recoveryKey, nil
}

func createCertificate(hostname string, ips []net.IP, signerCert []byte, signerKey crypto.PrivateKey) ([]byte, crypto.PrivateKey) {
	// TODO AB#875 cleanup
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

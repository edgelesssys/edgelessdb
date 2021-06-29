package db

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"
)

func splitHostPort(address, defaultPort string) (host, port string) {
	if address == "" {
		return "0.0.0.0", defaultPort
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address, defaultPort
	}
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = defaultPort
	}
	return
}

func toPEM(cert []byte, key crypto.PrivateKey) (pemCert, pemKey []byte, err error) {
	pemCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if len(pemCert) <= 0 {
		return nil, nil, errors.New("pem.EncodeToMemory failed")
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}

	pemKey = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if len(pemKey) <= 0 {
		return nil, nil, errors.New("pem.EncodeToMemory failed")
	}

	return
}

func createCertificate(commonName string) ([]byte, crypto.PrivateKey, error) {
	template := &x509.Certificate{
		SerialNumber:          &big.Int{},
		Subject:               pkix.Name{Organization: []string{"EDB root"}, CommonName: commonName},
		NotAfter:              time.Now().AddDate(10, 0, 0),
		DNSNames:              []string{commonName},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}
	return cert, priv, nil
}

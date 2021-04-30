package db

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
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

func generatePassword() (string, error) {
	var pw [30]byte
	if _, err := rand.Read(pw[:]); err != nil {
		return "", err
	}
	for i := range pw {
		pw[i] = pw[i]%('z'-'0'+1) + '0'
	}
	return string(pw[:]), nil
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

func createCertificate(commonName string) ([]byte, crypto.PrivateKey) {
	// TODO meaningful values
	template := &x509.Certificate{
		SerialNumber:          &big.Int{},
		Subject:               pkix.Name{Organization: []string{"EDB root"}, CommonName: commonName},
		NotAfter:              time.Now().AddDate(10, 0, 0),
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	return cert, priv
}

package db

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// EnvRootCertificate is the name of the environment variable holding the root certificate passed by Marblerun (created as a secret)
const EnvRootCertificate = "EDB_ROOT_CERT"

// EnvRootKey is the name of the environment variable holding the private key for root certificate passed by Marblerun (as a secret)
const EnvRootKey = "EDB_ROOT_KEY"

func setupCertificateFromMarblerun() ([]byte, crypto.PrivateKey, error) {
	// Retrieve root certificate and private key over environment, which Marblerun should pass through
	marbleSecretRootCert := os.Getenv(EnvRootCertificate)
	marbleSecretPrivKey := os.Getenv(EnvRootKey)

	// If some of them are empty or non-existant, abort. Secret definitions are required when running under Marblerun.
	if marbleSecretRootCert == "" || marbleSecretPrivKey == "" {
		return nil, nil, errors.New("edb did not retrieve secret definition for root certificate from marblerun")
	}

	// Decode root certificate PEM retrieved from env
	block, _ := pem.Decode([]byte(marbleSecretRootCert))
	if block == nil {
		return nil, nil, errors.New("failed to parse root certificate PEM from marblerun secret")
	}
	cert := block.Bytes

	// Check if we got a CA certificate
	certParsed, err := x509.ParseCertificate(cert)
	if err != nil {
		return nil, nil, err
	}
	if !certParsed.IsCA {
		return nil, nil, errors.New("root certificate passed from marblerun is not a CA")
	}

	// Decode private key PEM retrieved from env
	block, _ = pem.Decode([]byte(marbleSecretPrivKey))
	if block == nil {
		return nil, nil, errors.New("failed to parse private key PEM from marblerun secret")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	// Check if a reasonable key size was chosen in the secret definiton
	// For ed25519, Go enforces this whenever we use these keys and panics otherwise (though Marblerun should never allow this anyway)
	switch privKey := key.(type) {
	case *rsa.PrivateKey:
		if privKey.N.BitLen() < 2048 {
			return nil, nil, errors.New("rsa key retrieved from marblerun is too short")
		}
	case *ecdsa.PrivateKey:
		if privKey.Curve.Params().BitSize < 256 {
			return nil, nil, errors.New("ecdsa key retrieved from marblerun is too short")
		}
	}

	return cert, key, nil
}

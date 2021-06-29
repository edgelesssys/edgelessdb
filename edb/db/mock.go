package db

import (
	"crypto"
	"encoding/json"
)

// DatabaseMock is a Database mock.
type DatabaseMock struct {
	Man manifest
}

// GetCertificate gets the database certificate.
func (d *DatabaseMock) GetCertificate() ([]byte, crypto.PrivateKey) {
	cert, priv, err := createCertificate("")

	// The standard interface does not return an error as it just gets the certificate from the core.
	// Since the mock interface dynamically creates one, we assume this has to succeed otherwise we panic here.
	if err != nil {
		panic(err)
	}
	return cert, priv
}

// Initialize sets up a database according to the jsonManifest.
func (d *DatabaseMock) Initialize(jsonManifest []byte) error {
	if err := json.Unmarshal(jsonManifest, &d.Man); err != nil {
		return err
	}
	return nil
}

// Start starts the database.
func (d *DatabaseMock) Start() error {
	return nil
}

// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
func (d *DatabaseMock) GetManifestSignature() []byte {
	return nil
}

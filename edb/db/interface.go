package db

import "crypto"

// Database is a secure database that can be initialized by a manifest.
type Database interface {
	// GetCertificate gets the database certificate.
	GetCertificate() ([]byte, crypto.PrivateKey)
	// Initialize sets up a database according to the jsonManifest.
	Initialize(jsonManifest []byte) error
	// Start starts the database.
	Start() error
	// GetManifestSignature returns the signature of the manifest that has been used to initialize the database.
	GetManifestSignature() []byte
}

type manifest struct {
	SQL   []string
	CA    string
	Debug bool
}

package db

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCertificateFromMarblerun(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	os.Clearenv()
	defer os.Clearenv()

	// Generate certificate and key from our standalone implementation
	cert, key, err := createCertificate("Test CN")
	require.NoError(err)
	require.NotNil(cert)
	require.NotNil(key)

	// Marshal key & PEMify both
	keyPKCS8, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyPKCS8})

	// Now set both PEMs into the env as we would expect Marblerun to do with its manifests
	os.Setenv(EnvRootCertificate, string(certPEM))
	os.Setenv(EnvRootKey, string(keyPEM))

	// And check if we retrieve the same values back
	secretCert, secretKey, err := setupCertificateFromMarblerun()
	assert.NoError(err)
	assert.Equal(cert, secretCert)
	assert.Equal(key, secretKey)
}

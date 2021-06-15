package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"

	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	assert := assert.New(t)
	cert, key := createMockCertificate()
	core := newCoreWithMocks()

	assert.NoError(core.StartDatabase())

	jsonManifest := `
	{
		"sql": [
			"statement1",
			"statement2"
		],
		"ca": "cert",
		"recovery": "` + strings.ReplaceAll(cert, "\n", "\\n") + `"
	}`
	encRecKey, err := core.Initialize([]byte(jsonManifest))
	assert.NoError(err)
	assert.NotNil(encRecKey)
	recKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encRecKey, nil)
	assert.NoError(err)
	assert.Equal([]byte{3, 4, 5}, recKey)

	assert.Nil(core.GetManifestSignature())
}

func TestGetCertificateReport(t *testing.T) {
	assert := assert.New(t)
	core := newCoreWithMocks()

	assert.NoError(core.StartDatabase())

	_, quote, err := core.GetCertificateReport()
	assert.NoError(err)
	assert.Equal([]byte{2, 3, 4}, quote)
}

func TestEncryptRecoveryKey(t *testing.T) {
	assert := assert.New(t)
	cert, key := createMockCertificate()
	core := newCoreWithMocks()

	encRecKey, err := core.encryptRecoveryKey(cert)
	assert.NoError(err)
	recKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encRecKey, nil)
	assert.NoError(err)
	assert.Equal([]byte{3, 4, 5}, recKey)
}

func newCoreWithMocks() *Core {
	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	return NewCore(&rt, &db, false)
}

func createMockCertificate() (string, *rsa.PrivateKey) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	return string(pemCert), priv
}

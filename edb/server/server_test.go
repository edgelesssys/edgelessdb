package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/stretchr/testify/assert"
)

func TestManifest(t *testing.T) {
	assert := assert.New(t)

	const jsonManifest = `
		{
			"sql": [
				"statement1",
				"statement2"
			],
			"ca": "cert"
		}`

	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	core := core.NewCore(&rt, &db, false)
	mux := CreateServeMux(core)

	req := httptest.NewRequest("POST", "/manifest", strings.NewReader(jsonManifest))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(http.StatusOK, resp.Code)

	assert.Equal("cert", db.Man.CA)
	assert.Equal(2, len(db.Man.SQL))
}

func TestManifestRecovery(t *testing.T) {
	assert := assert.New(t)

	cert, key := createCertificate()

	jsonManifest := `
		{
			"recovery": "` + strings.ReplaceAll(cert, "\n", "\\n") + `"
		}`

	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	core := core.NewCore(&rt, &db, false)
	mux := CreateServeMux(core)

	req := httptest.NewRequest("POST", "/manifest", strings.NewReader(jsonManifest))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(http.StatusOK, resp.Code)

	ciphertext, err := base64.StdEncoding.DecodeString(resp.Body.String())
	assert.Nil(err)

	plaintext, err := rsa.DecryptOAEP(sha256.New(), nil, key, ciphertext, nil)
	assert.Nil(err)
	assert.Equal([]byte{3, 4, 5}, plaintext)
}

func createCertificate() (string, *rsa.PrivateKey) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	return string(pemCert), priv
}

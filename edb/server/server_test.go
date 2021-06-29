package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	core, db, _, _ := newCoreWithMocks()
	defer os.Unsetenv("EROCKSDB_MASTERKEY")
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
	require := require.New(t)

	cert, key, err := createMockRecoveryKey()
	require.NoError(err)

	jsonManifest := `
		{
			"recovery": "` + strings.ReplaceAll(cert, "\n", "\\n") + `"
		}`

	core, _, fs, tempPath := newCoreWithMocks()
	defer os.Unsetenv("EROCKSDB_MASTERKEY")
	mux := CreateServeMux(core)

	req := httptest.NewRequest("POST", "/manifest", strings.NewReader(jsonManifest))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(http.StatusOK, resp.Code)

	ciphertext, err := base64.StdEncoding.DecodeString(resp.Body.String())
	assert.NoError(err)

	plaintext, err := rsa.DecryptOAEP(sha256.New(), nil, key, ciphertext, nil)
	assert.NoError(err)

	// Get master key from encrypted file
	sealedKey, err := fs.ReadFile(filepath.Join(tempPath, "edb-persistence", "sealed_key"))
	assert.NoError(err)
	assert.Equal(sealedKey, plaintext)
}

func createMockRecoveryKey() (string, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}
	pubPKIX, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return "", nil, err
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})
	return string(pemKey), priv, nil
}

func newCoreWithMocks() (*core.Core, *db.DatabaseMock, afero.Afero, string) {
	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	tempPath, err := fs.TempDir("", "")
	if err != nil {
		panic(err)
	}
	cfg := core.Config{DataPath: tempPath}
	return core.NewCore(cfg, &rt, &db, fs, false), &db, fs, tempPath
}

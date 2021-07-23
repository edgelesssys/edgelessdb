/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pemKey, key, err := createMockRecoveryKey()
	require.NoError(err)
	core, _ := newCoreWithMocks()

	assert.NoError(core.StartDatabase())

	jsonManifest := `
	{
		"sql": [
			"statement1",
			"statement2"
		],
		"ca": "cert",
		"recovery": "` + strings.ReplaceAll(pemKey, "\n", "\\n") + `"
	}`
	encRecKey, err := core.Initialize([]byte(jsonManifest))
	assert.NoError(err)
	assert.NotNil(encRecKey)
	recKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encRecKey, nil)
	assert.NoError(err)
	assert.Equal(core.masterKey, recKey)

	assert.Nil(core.GetManifestSignature())
}

func TestGetCertificateReport(t *testing.T) {
	assert := assert.New(t)
	core, _ := newCoreWithMocks()

	assert.NoError(core.StartDatabase())

	_, quote, err := core.GetCertificateReport()
	assert.NoError(err)
	assert.Equal([]byte{2, 3, 4}, quote)
}

func TestEncryptRecoveryKey(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	pemKey, key, err := createMockRecoveryKey()
	require.NoError(err)
	core, _ := newCoreWithMocks()
	mockKey := []byte{3, 4, 5}

	encRecKey, err := core.encryptRecoveryKey(mockKey, pemKey)
	assert.NoError(err)
	recKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, encRecKey, nil)
	assert.NoError(err)
	assert.Equal(mockKey, recKey)
}

func newCoreWithMocks() (*Core, string) {
	rt := rt.RuntimeMock{}
	db := db.DatabaseMock{}
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	tempPath, err := fs.TempDir("", "")
	if err != nil {
		panic(err)
	}
	cfg := Config{DataPath: tempPath}
	return NewCore(cfg, &rt, &db, fs, false), tempPath
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

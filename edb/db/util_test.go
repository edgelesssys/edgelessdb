package db

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitHostPort(t *testing.T) {
	assert := assert.New(t)
	const defport = "defaultPort"

	h, p := splitHostPort("", "")
	assert.Equal("0.0.0.0", h)
	assert.Equal("", p)

	h, p = splitHostPort("", defport)
	assert.Equal("0.0.0.0", h)
	assert.Equal(defport, p)

	h, p = splitHostPort("addr", defport)
	assert.Equal("addr", h)
	assert.Equal(defport, p)

	h, p = splitHostPort(":", defport)
	assert.Equal("0.0.0.0", h)
	assert.Equal(defport, p)

	h, p = splitHostPort("addr:", defport)
	assert.Equal("addr", h)
	assert.Equal(defport, p)

	h, p = splitHostPort(":port", defport)
	assert.Equal("0.0.0.0", h)
	assert.Equal("port", p)

	h, p = splitHostPort("addr:port", defport)
	assert.Equal("addr", h)
	assert.Equal("port", p)
}

func TestGeneratePassword(t *testing.T) {
	assert := assert.New(t)

	p1, err := generatePassword()
	assert.Nil(err)
	assert.Equal(30, len(p1))

	p2, err := generatePassword()
	assert.Nil(err)
	assert.Equal(30, len(p2))

	assert.NotEqual(p1, p2)
}

func TestToPEM(t *testing.T) {
	assert := assert.New(t)

	cert, key := createCertificate("")
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	assert.Nil(err)

	pemCert, pemKey, err := toPEM(cert, key)
	assert.Nil(err)

	block, _ := pem.Decode(pemCert)
	assert.NotNil(block)
	assert.Equal(cert, block.Bytes)

	block, _ = pem.Decode(pemKey)
	assert.NotNil(block)
	assert.Equal(keyBytes, block.Bytes)
}

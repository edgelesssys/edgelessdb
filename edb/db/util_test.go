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

func TestToPEM(t *testing.T) {
	assert := assert.New(t)

	cert, key, err := createCertificate("")
	assert.NoError(err)
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

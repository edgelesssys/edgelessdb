package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateCertificateSerialNumber(t *testing.T) {
	assert := assert.New(t)
	serialNumber, err := GenerateCertificateSerialNumber()

	assert.NoError(err)
	assert.NotNil(serialNumber)
}

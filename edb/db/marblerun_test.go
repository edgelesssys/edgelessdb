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

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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillConfigFromEnvironment(t *testing.T) {
	config := Config{
		DataPath:           "data",
		DatabaseAddress:    "127.0.0.1",
		APIAddress:         "127.0.0.1:8080",
		CertificateDNSName: "localhost",
	}

	assert := assert.New(t)
	require := require.New(t)
	os.Clearenv()
	defer os.Clearenv() // reset environment when exiting the test

	// Empty
	newConfig := FillConfigFromEnvironment(config)
	assert.Equal(config, newConfig)

	// Partial
	require.NoError(os.Setenv(EnvAPIAddress, "1.2.3.4:1234"))

	newConfig = FillConfigFromEnvironment(config)
	assert.Equal("1.2.3.4:1234", newConfig.APIAddress)
	assert.Equal("127.0.0.1", newConfig.DatabaseAddress) // old value

	// Full
	require.NoError(os.Setenv(EnvAPIAddress, "1.2.3.4:1234"))
	require.NoError(os.Setenv(EnvDataPath, "edbTestDataPath"))
	require.NoError(os.Setenv(EnvDatabaseAddress, "1.2.3.4"))
	require.NoError(os.Setenv(EnvCertificateDNSName, "mytest-cn"))

	newConfig = FillConfigFromEnvironment(config)
	assert.Equal("1.2.3.4:1234", newConfig.APIAddress)
	assert.Equal("edbTestDataPath", newConfig.DataPath)
	assert.Equal("1.2.3.4", newConfig.DatabaseAddress)
	assert.Equal("mytest-cn", newConfig.CertificateDNSName)
}

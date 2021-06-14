package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillConfigFromEnvironment(t *testing.T) {
	config := Config{
		DataPath:              "data",
		DatabaseAddress:       "127.0.0.1",
		APIAddress:            "127.0.0.1:8080",
		CertificateCommonName: "localhost",
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
	require.NoError(os.Setenv(EnvCertificateCommonName, "mytest-cn"))

	newConfig = FillConfigFromEnvironment(config)
	assert.Equal("1.2.3.4:1234", newConfig.APIAddress)
	assert.Equal("edbTestDataPath", newConfig.DataPath)
	assert.Equal("1.2.3.4", newConfig.DatabaseAddress)
	assert.Equal("mytest-cn", newConfig.CertificateCommonName)
}

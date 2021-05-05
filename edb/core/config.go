package core

import (
	"encoding/json"
	"io/ioutil"
)

// Config is an EDB config.
type Config struct {
	DataPath              string `json:",omitempty"`
	DatabaseAddress       string `json:",omitempty"`
	APIAddress            string `json:",omitempty"`
	CertificateCommonName string `json:",omitempty"`
}

// ReadConfig reads the config from a file. Defaults will be used for undefined values in the file.
func ReadConfig(filename string, defaults Config) (Config, error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(configBytes, &defaults); err != nil {
		return Config{}, err
	}
	return defaults, nil
}

// +build !enclave

package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/edgelesssys/edb/edb/core"
)

func main() {
	config := core.Config{
		DataPath:              "data",
		DatabaseAddress:       "127.0.0.1",
		APIAddress:            "127.0.0.1:8080",
		CertificateCommonName: "localhost",
	}

	internalPath, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(internalPath)

	run(config, internalPath, "127.0.0.1")
}

func hostPath(path string) string {
	result, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return result
}

type runtime struct{}

func (runtime) GetRemoteReport(reportData []byte) ([]byte, error) {
	return nil, errors.New("GetRemoteReport: not running in an enclave")
}

func (runtime) GetProductSealKey() ([]byte, error) {
	return nil, errors.New("GetProductSealKey: not running in an enclave")
}

// +build enclave

package main

import (
	"os"
	"path/filepath"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/ego/enclave"
)

func main() {
	config := core.Config{
		DataPath:              "data",
		APIAddress:            ":8080",
		CertificateCommonName: "localhost",
	}

	internalPath := "/tmp/edb"
	if err := os.Mkdir(internalPath, 0); err != nil {
		panic(err)
	}

	//TODO AB#906
	//run(config, internalPath, "255.0.0.1")
	run(config, internalPath, "127.0.0.1")
}

func hostPath(path string) string {
	if !filepath.IsAbs(path) {
		cwd := os.Getenv("EDG_CWD")
		if cwd == "" {
			panic("cwd")
		}
		path = filepath.Join(cwd, path)
	}
	return filepath.Join(filepath.FromSlash("/edg"), "hostfs", filepath.Clean(path))
}

type runtime struct{}

func (runtime) GetRemoteReport(reportData []byte) ([]byte, error) {
	return enclave.GetRemoteReport(reportData)
}

func (runtime) GetProductSealKey() ([]byte, error) {
	key, _, err := enclave.GetProductSealKey()
	return key, err
}

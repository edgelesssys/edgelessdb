// +build enclave

package main

import (
	"flag"
	"os"
	"path/filepath"
	"syscall"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/ego/enclave"
)

func main() {
	configFilename := flag.String("c", "", "config file")
	flag.Parse()

	config := core.Config{
		DataPath:              "data",
		APIAddress:            ":8080",
		CertificateCommonName: "localhost",
	}

	if *configFilename != "" {
		var err error
		config, err = core.ReadConfig(hostPath(*configFilename), config)
		if err != nil {
			panic(err)
		}
	}

	internalPath := "/tmp/edb"
	if err := os.Mkdir(internalPath, 0); err != nil {
		panic(err)
	}

	// mount rocksdb dir from hostfs
	absDataPath := enclaveAbsPath(config.DataPath)
	if err := os.MkdirAll(hostPath(absDataPath), 0700); err != nil {
		panic(err)
	}
	if err := syscall.Mount(filepath.Join(absDataPath, "#rocksdb"), "/data/#rocksdb", "oe_host_file_system", 0, ""); err != nil {
		panic(err)
	}
	config.DataPath = "/data"

	run(config, internalPath, "255.0.0.1")
}

func enclaveAbsPath(path string) string {
	if !filepath.IsAbs(path) {
		cwd := os.Getenv("EDG_CWD")
		if cwd == "" {
			panic("cwd")
		}
		path = filepath.Join(cwd, path)
	}
	return path
}

func hostPath(path string) string {
	path = enclaveAbsPath(path)
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

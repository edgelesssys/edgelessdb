// +build enclave

package main

import (
	"flag"
	"os"
	"path/filepath"
	"syscall"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/ego/enclave"
	"github.com/edgelesssys/marblerun/marble/premain"
)

func main() {
	runAsMarble := flag.Bool("marble", false, "Run edb with Marblerun")
	flag.Parse()

	if *runAsMarble {
		// Contact Marblerun to provision edb
		if err := premain.PreMainEgo(); err != nil {
			panic(err)
		}
	}

	config := core.Config{
		DataPath:              "data",
		APIAddress:            ":8080",
		CertificateCommonName: "localhost",
		Debug:                 false,
		LogDir:                "",
	}

	// Load config parameters from environment variables
	config = core.FillConfigFromEnvironment(config)

	internalPath := "/tmp/edb"
	if err := os.Mkdir(internalPath, 0); err != nil {
		panic(err)
	}

	// mount logDir from hostfs if set
	if len(config.LogDir) > 0 {
		absLogPath := enclaveAbsPath(config.LogDir)
		if err := os.MkdirAll(hostPath(absLogPath), 0700); err != nil {
			panic(err)
		}
		if err := syscall.Mount(absLogPath, "/log", "oe_host_file_system", 0, ""); err != nil {
			panic(err)
		}
		config.LogDir = "/log"
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

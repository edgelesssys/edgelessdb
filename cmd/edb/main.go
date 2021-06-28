// +build !enclave

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/marblerun/marble/premain"
)

func main() {
	runAsMarble := flag.Bool("marble", false, "Run edb with Marblerun")
	flag.Parse()

	if *runAsMarble {
		// Contact Marblerun to provision edb
		if err := premain.PreMainMock(); err != nil {
			panic(err)
		}
	}

	config := core.Config{
		DataPath:              "data",
		DatabaseAddress:       "127.0.0.1",
		APIAddress:            "127.0.0.1:8080",
		CertificateCommonName: "localhost",
		Debug:                 false,
		LogDir:                "",
	}

	// Load config parameters from environment variables
	config = core.FillConfigFromEnvironment(config)

	internalPath, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(internalPath)

	config.DataPath = hostPath(config.DataPath)

	// Warn user this is not a trustful setup at all!
	fmt.Println("edb is running in non-enclave mode without Marblerun.")
	fmt.Println("This means edb will save the encryption key used IN PLAINTEXT on the disk.")
	fmt.Println("THIS IS OBVIOUSLY NOT SECURE AT ALL FOR PRODUCTION!")
	fmt.Println("Only ever use non-enclave mode for testing, please...")
	if err := os.MkdirAll(path.Join(hostPath(config.DataPath), core.PersistenceDir), 0700); err != nil && !os.IsExist(err) {
		panic(err)
	}

	run(config, *runAsMarble, internalPath, "127.0.0.1")
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
	return make([]byte, 16), nil
}

func (runtime) IsEnclave() bool {
	return false
}

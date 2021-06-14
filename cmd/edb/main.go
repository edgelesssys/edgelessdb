// +build !enclave

package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
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
	return make([]byte, 16), nil
}

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/server"
)

func main() {
	configFilename := flag.String("c", "", "")
	flag.Parse()

	cfg := struct {
		DataPath        string
		DatabaseAddress string
		APIAddress      string
	}{
		"data",
		"127.0.0.1",
		"127.0.0.1:8080",
	}

	if *configFilename != "" {
		config, err := ioutil.ReadFile(*configFilename)
		if err != nil || json.Unmarshal(config, &cfg) != nil {
			panic("config")
		}
	}
	cfg.DataPath, _ = filepath.Abs(cfg.DataPath)
	os.MkdirAll(cfg.DataPath, 0700)

	rt := runtime{}
	internalPath, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(internalPath)
	db, err := db.NewMariadb(internalPath, cfg.DataPath, "127.0.0.1", cfg.DatabaseAddress, "localhost", mariadbd{})
	if err != nil {
		panic(err)
	}

	core := core.NewCore(rt, db)
	mux := server.CreateServeMux(core)
	if err := core.StartDatabase(); err != nil {
		panic(err)
	}
	server.RunServer(mux, cfg.APIAddress, core.GetTLSConfig())
}

type runtime struct{}

func (runtime) GetRemoteReport(reportData []byte) ([]byte, error) {
	return nil, errors.New("GetRemoteReport: not running in an enclave")
}

func (runtime) GetProductSealKey() ([]byte, error) {
	return nil, errors.New("GetProductSealKey: not running in an enclave")
}

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/rt"
	"github.com/edgelesssys/edb/edb/server"
	"github.com/edgelesssys/edb/tidb"
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

	rt := rt.RuntimeMock{}
	internalPath, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(internalPath)
	launcher := tidb.Launcher{ConfigPath: filepath.Join(internalPath, "tidbcfg")}
	db, err := db.NewTidb(internalPath, cfg.DataPath, "127.0.0.1", cfg.DatabaseAddress, "localhost", &launcher)
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

package main

import (
	"flag"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/server"
)

func run(config core.Config, internalPath string, internalAddress string) {
	configFilename := flag.String("c", "", "config file")
	flag.Parse()

	cfg := config
	if *configFilename != "" {
		var err error
		cfg, err = core.ReadConfig(hostPath(*configFilename), config)
		if err != nil {
			panic(err)
		}
	}

	db, err := db.NewMariadb(internalPath, hostPath(cfg.DataPath), internalAddress, cfg.DatabaseAddress, cfg.CertificateCommonName, mariadbd{})
	if err != nil {
		panic(err)
	}

	core := core.NewCore(runtime{}, db)
	mux := server.CreateServeMux(core)
	if err := core.StartDatabase(); err != nil {
		panic(err)
	}
	server.RunServer(mux, cfg.APIAddress, core.GetTLSConfig())
}

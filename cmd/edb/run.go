package main

import (
	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/server"
)

func run(cfg core.Config, internalPath string, internalAddress string) {
	db, err := db.NewMariadb(internalPath, cfg.DataPath, internalAddress, cfg.DatabaseAddress, cfg.CertificateCommonName, mariadbd{})
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

package main

import (
	"encoding/hex"
	"os"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/server"
)

func run(cfg core.Config, isMarble bool, internalPath string, internalAddress string) {
	db, err := db.NewMariadb(internalPath, cfg.DataPath, internalAddress, cfg.DatabaseAddress, cfg.CertificateCommonName, cfg.LogDir, cfg.Debug, mariadbd{})
	if err != nil {
		panic(err)
	}

	var rt runtime

	// set product key as erocks masterkey
	key, err := rt.GetProductSealKey()
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("EROCKSDB_MASTERKEY", hex.EncodeToString(key)); err != nil {
		panic(err)
	}

	core := core.NewCore(rt, db, isMarble)
	mux := server.CreateServeMux(core)
	if err := core.StartDatabase(); err != nil {
		panic(err)
	}
	server.RunServer(mux, cfg.APIAddress, core.GetTLSConfig())
}

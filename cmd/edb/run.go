package main

import (
	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/edgelesssys/edb/edb/server"
	"github.com/fatih/color"
	"github.com/spf13/afero"
)

func run(cfg core.Config, isMarble bool, internalPath string, internalAddress string) {
	db, err := db.NewMariadb(internalPath, cfg.DataPath, internalAddress, cfg.DatabaseAddress, cfg.CertificateCommonName, cfg.LogDir, cfg.Debug, isMarble, mariadbd{})
	if err != nil {
		panic(err)
	}

	var rt runtime
	fs := afero.Afero{Fs: afero.NewOsFs()}
	core := core.NewCore(cfg, rt, db, fs, isMarble)

	mux := server.CreateServeMux(core)
	if !core.IsRecovering() {
		if err := core.StartDatabase(); err != nil {
			panic(err)
		}
	} else {
		color.Red("edb failed to retrieve the database encryption key and has entered recovery mode.")
		color.Red("You can use the /recover API endpoint to upload the recovery data which was generated when the manifest has been initialized originally.")
		color.Red("For more information, consult the documentation.") // TODO: Add URL to our documentation
	}
	server.RunServer(mux, cfg.APIAddress, core.GetTLSConfig())
}

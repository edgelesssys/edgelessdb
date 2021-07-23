/* Copyright (c) Edgeless Systems GmbH

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1335  USA */

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

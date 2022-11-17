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
	"github.com/edgelesssys/edgelessdb/edb/core"
	"github.com/edgelesssys/edgelessdb/edb/db"
	"github.com/edgelesssys/edgelessdb/edb/server"
	"github.com/fatih/color"
	"github.com/spf13/afero"
)

func run(cfg core.Config, isMarble bool, internalPath string, internalAddress string) {
	var rt executionEnv

	// There are quite a few MariaDB and RocksDB helper threads in addition to pool threads. Let's be rather generous here.
	maxPoolThreads := rt.GetNumTCS() - 32

	db, err := db.NewMariadb(internalPath, cfg.DataPath, internalAddress, cfg.DatabaseAddress, cfg.CertificateDNSName, cfg.LogDir, cfg.Debug, isMarble, mariadbd{}, maxPoolThreads)
	if err != nil {
		panic(err)
	}

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
		color.Red("For more information, visit: https://edglss.cc/doc-edb-recovery")

		// Generate quote for temporary certificate in recovery mode
		// Should not be able to panic as GenerateReport only returns errors in Marble mode, however recovery is not available in Marble mode.
		if err := core.GenerateReport(); err != nil {
			panic(err)
		}
	}
	server.RunServer(mux, cfg.APIAddress, core.GetTLSConfig())
}

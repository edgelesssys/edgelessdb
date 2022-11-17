// +build enclave

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
	"flag"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/edgelesssys/edgelessdb/edb/core"
	"github.com/edgelesssys/edgelessdb/edb/rt"
	"github.com/edgelesssys/ego/enclave"
	"github.com/edgelesssys/marblerun/marble/premain"
)

// Don't touch! Automatically injected at build-time.
var version = "0.0.0"
var gitCommit = "0000000000000000000000000000000000000000"

const internalPath = "/tmp/edb" // supposed to be mounted in emain.cpp

func main() {
	rt.Log.Printf("EdgelessDB v%v (%v)\n", version, gitCommit)

	runAsMarble := flag.Bool("marble", false, "Run edb with Marblerun")
	flag.Parse()

	if *runAsMarble {
		// Contact Marblerun to provision edb
		if err := premain.PreMainEgo(); err != nil {
			panic(err)
		}
	}

	config := core.Config{
		DataPath:           "data",
		APIAddress:         ":8080",
		CertificateDNSName: "localhost",
		Debug:              false,
		LogDir:             "",
	}

	// Load config parameters from environment variables
	config = core.FillConfigFromEnvironment(config)

	if err := os.Mkdir(internalPath, 0); err != nil {
		panic(err)
	}

	// mount logDir from hostfs if set
	if len(config.LogDir) > 0 {
		absLogPath := enclaveAbsPath(config.LogDir)
		if err := os.MkdirAll(hostPath(absLogPath), 0700); err != nil {
			panic(err)
		}
		if err := syscall.Mount(absLogPath, "/log", "oe_host_file_system", 0, ""); err != nil {
			panic(err)
		}
		config.LogDir = "/log"
	}

	// mount rocksdb dir from hostfs
	absDataPath := enclaveAbsPath(config.DataPath)
	if err := os.MkdirAll(hostPath(absDataPath), 0700); err != nil {
		panic(err)
	}
	if err := syscall.Mount(filepath.Join(absDataPath, "#rocksdb"), "/data/#rocksdb", "oe_host_file_system", 0, ""); err != nil {
		panic(err)
	}

	// Create to store sealing key
	if !*runAsMarble {
		if err := syscall.Mount(filepath.Join(absDataPath, core.PersistenceDir), filepath.Join("/data", core.PersistenceDir), "oe_host_file_system", 0, ""); err != nil {
			panic(err)
		}
		if err := os.Mkdir(path.Join(hostPath(absDataPath), core.PersistenceDir), 0700); err != nil && !os.IsExist(err) {
			panic(err)
		}
	}

	config.DataPath = "/data"

	run(config, *runAsMarble, internalPath, "255.0.0.1")
}

func enclaveAbsPath(path string) string {
	if !filepath.IsAbs(path) {
		cwd := os.Getenv("EDG_CWD")
		if cwd == "" {
			panic("cwd")
		}
		path = filepath.Join(cwd, path)
	}
	return path
}

func hostPath(path string) string {
	path = enclaveAbsPath(path)
	return filepath.Join(filepath.FromSlash("/edg"), "hostfs", filepath.Clean(path))
}

type executionEnv struct{}

func (executionEnv) GetRemoteReport(reportData []byte) ([]byte, error) {
	return enclave.GetRemoteReport(reportData)
}

func (executionEnv) GetProductSealKey() ([]byte, error) {
	key, _, err := enclave.GetProductSealKey()
	return key, err
}

func (executionEnv) IsEnclave() bool {
	return true
}

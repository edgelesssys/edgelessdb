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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgelesssys/edb/edb/core"
	"github.com/edgelesssys/edb/edb/db"
	"github.com/fatih/color"
)

const errRocksDBInitFailed = "Plugin 'ROCKSDB' registration as a STORAGE ENGINE failed."

func exit(status int) {
	determineError() // Print more specific error whenever we can detect one
	color.Red("edb has exited unexpectedly (exit code: %d).", status)
	os.Exit(status)
}

func determineError() {
	// Try to read error log either from internal memfs (debug = off or no path specified) or from a specified path (debug = on + path specified)
	var errorLogBasePath string
	var pointUserToDebugLog bool

	if os.Getenv(core.EnvDebug) == "" {
		errorLogBasePath = internalPath
	} else {
		logDir := os.Getenv(core.EnvLogDir)
		if logDir != "" {
			errorLogBasePath = hostPath(logDir)
			pointUserToDebugLog = true
		} else {
			// Cannot determine, as this was printed to stderr. User needs to look into the terminal by themself.
			return
		}
	}

	errorLogBytes, err := ioutil.ReadFile(filepath.Join(errorLogBasePath, db.FilenameErrorLog))
	if err != nil {
		return
	}
	errorLog := string(errorLogBytes)
	if strings.Contains(errorLog, errRocksDBInitFailed) {
		fmt.Fprint(os.Stderr, errorLog) // Always print error log in this case, as we expect that a failed initialization should not leak any sensitive data
		color.Red("eRocksDB failed to initialize correctly.")
		color.Red("This likely failed due to an incorrect key being used to decrypt the database or the database being corrupted.")
		color.Red("Make sure you run edb on the same machine as it was initialized on.")
	}
	if pointUserToDebugLog {
		cleanPath := strings.TrimPrefix(filepath.Join(errorLogBasePath, db.FilenameErrorLog), "/edg/hostfs")
		color.Red("You can find the error log at: %s", cleanPath)
	}
}

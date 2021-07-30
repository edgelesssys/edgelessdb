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

// extern int edgeless_exit_ensure_link;
import "C"
import "github.com/edgelesssys/edgelessdb/edb/rt"

//export invokemain
func invokemain() {
	// Save original stdout & stderr before we ever launch MariaDB, as MariaDB will redirect it later on
	if err := rt.SaveStdoutAndStderr(); err != nil {
		panic(err)
	}

	main()
}

//export edgeless_exit
func edgeless_exit(status C.int) {
	C.edgeless_exit_ensure_link = 1

	// Restore original stdout & stderr from MariaDB's redirection
	if err := rt.RestoreStdoutAndStderr(); err != nil {
		panic(err)
	}
	exit(int(status))
}

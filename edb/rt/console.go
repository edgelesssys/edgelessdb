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

package rt

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

var savedStdout int
var savedStderr int

// Log provides edb's logging functionality before we enter MariaDB or after we exited from there
var Log = log.New(os.Stdout, "[EDB] ", log.LstdFlags)

// SaveStdoutAndStderr saves the stdout/stderr outputs before we call into MariaDB
func SaveStdoutAndStderr() error {
	var err error

	savedStdout, err = syscall.Dup(syscall.Stdout)
	if err != nil {
		return fmt.Errorf("cannot save original stdout: %v", err)
	}
	savedStderr, err = syscall.Dup(syscall.Stderr)
	if err != nil {
		return fmt.Errorf("cannot save original stderr: %v", err)
	}

	return nil
}

// RestoreStdoutAndStderr restores the stdout/stderr which MariaDB redirected for its error log outputs after we returned from it
func RestoreStdoutAndStderr() error {
	if err := syscall.Dup2(savedStdout, syscall.Stdout); err != nil {
		return fmt.Errorf("cannot restore stdout from redirection: %v", err)
	}
	if err := syscall.Dup2(savedStderr, syscall.Stderr); err != nil {
		return fmt.Errorf("cannot restore stderr from redirection: %v", err)
	}

	return nil
}

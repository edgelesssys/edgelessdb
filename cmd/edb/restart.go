// +build !enclave

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

/*
#include <stdlib.h>
#include <unistd.h>

static char** _argv;

__attribute__((constructor)) static void init(int argc, char** argv) {
	_argv = argv;
}

static void restart() {
	execv("/proc/self/exe", _argv);
	abort();
}
*/
import "C"

import "os"

var initialWorkingDir = func() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}()

func (executionEnv) RestartHostProcess() {
	if err := os.Chdir(initialWorkingDir); err != nil {
		panic(err)
	}
	C.restart()
}

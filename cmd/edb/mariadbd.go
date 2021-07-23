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
#cgo LDFLAGS: -Wl,-unresolved-symbols=ignore-in-object-files
#include <unistd.h>
int edgeless_mysqld_main(int argc, char** argv);

static void waitUntilSet(volatile int* p) {
	do {
		usleep(10000);
	} while (!__atomic_load_n(p, __ATOMIC_SEQ_CST));
}

static void waitUntilStarted() {
	extern volatile int mysqld_server_started;
	waitUntilSet(&mysqld_server_started);
}

static void waitUntilListenInternalReady() {
	extern int edgeless_listen_internal_ready;
	waitUntilSet(&edgeless_listen_internal_ready);
}
*/
import "C"

type mariadbd struct{}

func (mariadbd) Main(cnfPath string) int {
	argv := []*C.char{C.CString("edb"), C.CString("--defaults-file=" + cnfPath), nil}
	return int(C.edgeless_mysqld_main(2, &argv[0]))
}

func (mariadbd) WaitUntilStarted() {
	C.waitUntilStarted()
}

func (mariadbd) WaitUntilListenInternalReady() {
	C.waitUntilListenInternalReady()
}

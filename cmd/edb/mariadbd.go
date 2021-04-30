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

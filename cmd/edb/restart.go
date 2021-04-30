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

func (runtime) RestartHostProcess() {
	if err := os.Chdir(initialWorkingDir); err != nil {
		panic(err)
	}
	C.restart()
}

package main

// extern int edgeless_exit_ensure_link;
import "C"
import "github.com/edgelesssys/edb/edb/rt"

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

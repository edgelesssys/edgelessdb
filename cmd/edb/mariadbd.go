package main

/*
#cgo LDFLAGS: -Wl,-unresolved-symbols=ignore-in-object-files
int edgeless_mysqld_main(int argc, char** argv);
*/
import "C"

type mariadbd struct{}

func (mariadbd) Main(cnfPath string) int {
	argv := []*C.char{C.CString("edb"), C.CString("--defaults-file=" + cnfPath), nil}
	return int(C.edgeless_mysqld_main(2, &argv[0]))
}

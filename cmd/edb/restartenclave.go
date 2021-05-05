// +build enclave

package main

// void ert_restart_host_process(void);
import "C"

func (runtime) RestartHostProcess() {
	C.ert_restart_host_process()
}

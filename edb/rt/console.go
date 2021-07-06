package rt

import (
	"fmt"
	"syscall"
)

var savedStdout int
var savedStderr int

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

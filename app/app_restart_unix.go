//go:build !windows

package app

import (
	"os"
	"syscall"
)

func init() {
	// Add SIGTERM to default shutdown signals on Unix
	defaultSigs = append(defaultSigs, syscall.SIGTERM)
}

// restartSignal returns the default restart signal on Unix (SIGUSR2).
func restartSignal() os.Signal {
	return syscall.SIGUSR2
}

// getSysProcAttr returns platform-specific process attributes for restart.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

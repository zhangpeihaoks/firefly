//go:build windows

package app

import (
	"os"
	"syscall"
)

// restartSignal returns nil on Windows — fd-passing restart is not supported.
// Windows applications should use external process managers for restarts.
func restartSignal() os.Signal {
	return nil
}

// getSysProcAttr returns nil on Windows since fd passing is unsupported.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

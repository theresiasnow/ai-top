//go:build !windows

package monitor

import "syscall"

func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func restartProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}

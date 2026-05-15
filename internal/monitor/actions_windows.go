//go:build windows

package monitor

import (
	"fmt"
	"os"
)

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func restartProcess(pid int) error {
	return fmt.Errorf("restart is not supported on windows")
}

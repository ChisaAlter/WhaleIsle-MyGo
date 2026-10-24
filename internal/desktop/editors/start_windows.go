//go:build windows

package editors

import (
	"os/exec"
	"syscall"
)

// start launches the editor detached from the shell's lifetime and never waits
// for it (Electron spawns with detached:true, stdio:'ignore', windowsHide:true).
func start(exe string, args []string) error {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

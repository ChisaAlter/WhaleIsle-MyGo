//go:build windows

package harness

import (
	"os/exec"
	"strconv"
	"syscall"
)

// createNoWindow keeps npx.cmd and node from opening a console window.
const createNoWindow = 0x08000000

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// terminate ends the child's process tree; Windows has no graceful signal
// for console-less children, so both modes use taskkill /T.
func terminate(cmd *exec.Cmd, force bool) error {
	if cmd.Process == nil {
		return nil
	}
	args := []string{"/PID", strconv.Itoa(cmd.Process.Pid), "/T"}
	if force {
		args = append(args, "/F")
	}
	kill := exec.Command("taskkill", args...)
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return kill.Run()
}

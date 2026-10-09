//go:build !windows

package harness

import (
	"os/exec"
	"syscall"
)

// detach puts the child in its own process group, so stopping it also stops
// the node processes npx and dsh start.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminate signals the child's process group; force uses SIGKILL.
func terminate(cmd *exec.Cmd, force bool) error {
	if cmd.Process == nil {
		return nil
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	return syscall.Kill(-cmd.Process.Pid, sig)
}

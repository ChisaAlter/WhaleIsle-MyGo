//go:build !windows

package phoneremote

import (
	"os/exec"
	"syscall"
)

type bridgeJob struct{}

func prepareBridgeCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func bindBridgeJob(*exec.Cmd) *bridgeJob { return nil }
func (*bridgeJob) close()                {}

func killBridgeTree(cmd *exec.Cmd, _ *bridgeJob) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

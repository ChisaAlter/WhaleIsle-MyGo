//go:build windows

package harness

import (
	"os/exec"
	"strconv"
	"whaleisle/internal/processutil"
)

func detach(cmd *exec.Cmd) {
	processutil.Background(cmd)
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
	processutil.Background(kill)
	return kill.Run()
}

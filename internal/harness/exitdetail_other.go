//go:build !windows

package harness

import (
	"errors"
	"os/exec"
	"strconv"
	"syscall"
)

// exitDetail extracts the exit code, or the signal that killed the child,
// behind err, which may wrap the process's wait error.
func exitDetail(err error) (code, signal string) {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return "", ""
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return "", ws.Signal().String()
	}
	return strconv.Itoa(ee.ExitCode()), ""
}

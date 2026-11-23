//go:build windows

package harness

import (
	"errors"
	"os/exec"
	"strconv"
)

// exitDetail extracts the exit code behind err, which may wrap the
// process's wait error. Windows has no signal concept for child exit.
func exitDetail(err error) (code, signal string) {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return "", ""
	}
	return strconv.Itoa(ee.ExitCode()), ""
}

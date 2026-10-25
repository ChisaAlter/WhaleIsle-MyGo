//go:build windows

package git

import (
	"os/exec"
	"whaleisle/internal/processutil"
)

// Git status reads run repeatedly in the desktop. Keep their console processes
// hidden even when the parent executable uses the Windows GUI subsystem.
func configureGitProcess(cmd *exec.Cmd) {
	processutil.Background(cmd)
}

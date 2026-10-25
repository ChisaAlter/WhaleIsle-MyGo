//go:build !windows

package git

import "os/exec"

func configureGitProcess(cmd *exec.Cmd) {}

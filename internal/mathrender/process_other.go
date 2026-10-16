//go:build !windows

package mathrender

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}

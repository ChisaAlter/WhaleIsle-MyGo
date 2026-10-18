//go:build !windows

package petwatch

import "os/exec"

func hideScanner(*exec.Cmd) {}

//go:build !windows

package petgrowth

import "os/exec"

func hideScanner(*exec.Cmd) {}

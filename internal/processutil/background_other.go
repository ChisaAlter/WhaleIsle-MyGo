//go:build !windows

// Package processutil configures non-interactive desktop child processes.
package processutil

import "os/exec"

// Background leaves non-Windows process attributes unchanged.
func Background(cmd *exec.Cmd) {}

// BackgroundTree leaves non-Windows process attributes unchanged.
func BackgroundTree(cmd *exec.Cmd) {}

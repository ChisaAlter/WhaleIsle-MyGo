//go:build windows

// Package processutil configures non-interactive desktop child processes.
package processutil

import (
	"os/exec"
	"syscall"
)

// Background prevents a console window from being created for a background
// command. Redirecting standard IO or hiding the parent does not do this.
// Call before Start/Run; do not use for interactive terminals or GUI handoffs.
// Existing process attributes (tokens, handles and creation flags) are preserved.
func Background(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}

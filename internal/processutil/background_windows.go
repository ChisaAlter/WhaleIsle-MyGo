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

// BackgroundTree starts a runtime with a console that is hidden before its first
// instruction. Ordinary console descendants inherit this invisible console even
// when a Node package does not pass windowsHide. CREATE_NO_WINDOW suppresses only
// the immediate process's console and leaves those descendants free to allocate
// new visible windows. Standard streams remain the caller's pipes/files.
// Use for command hosts/workers, never interactive terminals or GUI handoffs.
func BackgroundTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	// Windows ignores CREATE_NO_WINDOW with CREATE_NEW_CONSOLE. Clear both it
	// and DETACHED_PROCESS explicitly instead of leaving conflicting flags.
	cmd.SysProcAttr.CreationFlags &^= 0x08000000 | 0x00000008
	cmd.SysProcAttr.CreationFlags |= 0x00000010 // CREATE_NEW_CONSOLE, SW_HIDE
}

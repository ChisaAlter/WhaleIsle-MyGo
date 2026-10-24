//go:build !windows

package editors

import "os/exec"

// start launches the editor without waiting for it; output is discarded so a
// long-running editor never holds the shell's pipes (Electron spawns with
// detached:true, stdio:'ignore').
func start(exe string, args []string) error {
	cmd := exec.Command(exe, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

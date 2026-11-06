//go:build windows

package harness

import (
	"os/exec"
	"strconv"
	"whaleisle/internal/processutil"
)

func detach(cmd *exec.Cmd) {
	processutil.BackgroundTree(cmd)
}

// terminate force-ends the child's process tree; force is ignored. The Host
// runs in a hidden console with no window to receive taskkill's close request,
// so a plain taskkill never ends it and would only hold Stop for stopGrace.
// Electron's killTree (reference/electron/src/main/dsh.js) also uses /T /F.
func terminate(cmd *exec.Cmd, _ bool) error {
	if cmd.Process == nil {
		return nil
	}
	kill := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
	processutil.Background(kill)
	return kill.Run()
}

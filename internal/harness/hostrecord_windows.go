//go:build windows

package harness

import (
	"os/exec"
	"strconv"

	"golang.org/x/sys/windows"

	"whaleisle/internal/processutil"
)

// processImage names the executable behind pid. A pid that is gone reads as
// empty; Windows always answers the question, so known is true.
func processImage(pid int) (string, bool) {
	if pid <= 0 {
		return "", true
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", true
	}
	defer windows.CloseHandle(handle)
	var name [32768]uint16
	size := uint32(len(name))
	if err := windows.QueryFullProcessImageName(handle, 0, &name[0], &size); err != nil {
		return "", true
	}
	return windows.UTF16ToString(name[:size]), true
}

// reapHostProcess ends the Host's process tree with the same call the supervisor
// uses for a child it owns: the Host runs in a hidden console, so only a forced
// taskkill ends it.
func reapHostProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	kill := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	processutil.Background(kill)
	return kill.Run()
}

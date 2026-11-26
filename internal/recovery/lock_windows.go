//go:build windows

package recovery

import "golang.org/x/sys/windows"

const stillActiveExitCode = 259

func lockProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		if err == windows.ERROR_INVALID_PARAMETER {
			return false
		}
		// Access denied or other error: fail closed and treat owner as alive.
		return true
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err == nil {
		return code == stillActiveExitCode
	}
	status, err := windows.WaitForSingleObject(h, 0)
	return err != nil || status != windows.WAIT_OBJECT_0
}

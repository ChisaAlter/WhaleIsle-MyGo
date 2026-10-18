package windows

import "errors"

// Shell_NotifyIcon returns a BOOL. Do not infer success from a last-error value:
// only its nonzero return authorizes registering the icon as available.
func trayAddResult(result uintptr) (bool, error) {
	if result == 0 {
		return false, errors.New("mygo: Shell_NotifyIconW could not add the tray icon")
	}
	return true, nil
}

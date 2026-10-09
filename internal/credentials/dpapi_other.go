//go:build !windows

package credentials

import "errors"

// DPAPI only exists on Windows, where the Electron shell also runs; other
// platforms get a clear error instead of a build failure.
func dpapiProtect([]byte) ([]byte, error) {
	return nil, errors.New("credentials: DPAPI is only available on Windows")
}

func dpapiUnprotect([]byte) ([]byte, error) {
	return nil, errors.New("credentials: DPAPI is only available on Windows")
}

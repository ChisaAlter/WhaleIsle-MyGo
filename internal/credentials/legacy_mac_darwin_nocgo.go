//go:build darwin && !cgo

package credentials

import "errors"

func decryptSafeStorage(string, []byte) ([]byte, error) {
	return nil, errors.New("credentials: legacy macOS safeStorage requires Keychain support (cgo)")
}

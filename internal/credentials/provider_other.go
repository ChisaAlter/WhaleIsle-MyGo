//go:build !darwin

package credentials

import "runtime"

func platformCredentialProvider() credentialProvider {
	return credentialProvider{
		version:   EnvelopeVersion,
		backend:   "Windows DPAPI + AES-256-GCM (safeStorage-v1)",
		supported: runtime.GOOS == "windows",
		key: func(dir string, create bool) ([]byte, error) {
			return loadOrCreateLocalStateKey(dir, create)
		},
	}
}

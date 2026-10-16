//go:build !darwin

package credentials

import "runtime"

func platformCredentialProvider() credentialProvider {
	return credentialProvider{EnvelopeVersion, "Windows DPAPI + AES-256-GCM (safeStorage-v1)", runtime.GOOS == "windows", func(dir string, _ bool) ([]byte, error) { return loadLocalStateKey(dir) }}
}

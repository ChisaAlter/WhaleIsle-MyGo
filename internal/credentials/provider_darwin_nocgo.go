//go:build darwin && !cgo

package credentials

import "errors"

func platformCredentialProvider() credentialProvider {
	return credentialProvider{KeychainEnvelopeVersion, "macOS Keychain (requires cgo)", false, func(string, bool) ([]byte, error) { return nil, errors.New("credentials: macOS Keychain requires cgo") }}
}

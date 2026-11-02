//go:build darwin && !cgo

package credentials

func decryptSafeStorage(_ string, blob []byte) ([]byte, error) {
	return openLegacyMacWithKeychain(blob, func() ([]byte, error) {
		a, err := loadKeychainAPI()
		if err != nil {
			return nil, err
		}
		// Query only the original application's documented identity. No writes,
		// enumeration, fallback application identities or ACL changes.
		return a.read("Whale Isle Safe Storage", "Whale Isle", 1, 4096)
	})
}

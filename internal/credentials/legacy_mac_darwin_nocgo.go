//go:build darwin && !cgo

package credentials

func decryptSafeStorage(_ string, blob []byte) ([]byte, error) {
	return openLegacyMacWithKeychain(blob, func() ([]byte, error) {
		a, err := loadKeychainAPI()
		if err != nil {
			return nil, err
		}
		return a.read(macSafeStorageService, macSafeStorageAccount, 1, 4096)
	})
}

func macSafeStorageSeal(_ string, plaintext []byte, create bool) (string, []byte, error) {
	a, err := loadKeychainAPI()
	if err != nil {
		return "", nil, err
	}
	password, err := getMacSafeStoragePassword(create,
		func() ([]byte, error) { return a.read(macSafeStorageService, macSafeStorageAccount, 1, 4096) },
		func(password []byte) error {
			return a.addPassword(macSafeStorageService, macSafeStorageAccount, password)
		})
	defer clear(password)
	if err != nil {
		return "", nil, err
	}
	payload, err := sealLegacyMac(plaintext, password)
	return EnvelopeVersion, payload, err
}

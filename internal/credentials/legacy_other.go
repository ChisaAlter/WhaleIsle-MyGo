//go:build !darwin

package credentials

func decryptSafeStorage(dir string, blob []byte) ([]byte, error) {
	key, err := loadLocalStateKey(dir)
	if err != nil {
		return nil, err
	}
	return open(key, blob)
}

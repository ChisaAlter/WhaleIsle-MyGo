package credentials

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
)

// KeychainEnvelopeVersion distinguishes the native macOS format from
// Electron's platform-specific safeStorage format; it is not Electron CBC.
const KeychainEnvelopeVersion = "macOS-Keychain-AES256GCM-v1"

var errKeychainNotFound = errors.New("credentials: Keychain key not found")
var errKeychainDuplicate = errors.New("credentials: Keychain key already exists")

type keychainBackend interface {
	get(account string) ([]byte, error)
	add(account string, key []byte) error
}

// The directory scopes a key to a single profile without storing its path in
// Keychain. Moving a profile requires migration; never silently replace a key.
func keychainKey(backend keychainBackend, dir string, create bool) ([]byte, error) {
	path, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	account := fmt.Sprintf("profile-%x", sha256.Sum256([]byte(filepath.Clean(path))))
	key, err := backend.get(account)
	if errors.Is(err, errKeychainNotFound) && create {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			clear(key)
			return nil, err
		}
		err = backend.add(account, key)
		if errors.Is(err, errKeychainDuplicate) {
			clear(key)
			key, err = backend.get(account)
		}
	}
	if err != nil {
		clear(key)
		return nil, err
	}
	if len(key) != 32 {
		clear(key)
		return nil, errors.New("credentials: invalid Keychain key length")
	}
	return key, nil
}

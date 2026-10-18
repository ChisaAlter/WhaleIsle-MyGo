package credentials

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

// Electron v44.3.0 restores this synchronous Chromium format in
// patches/chromium/revert_oscrypt_remove_sync_backend.patch. This is a read-only
// decoder, not a new encryption format. Unknown prefixes never become plaintext.
func openLegacyMac(blob, password []byte) ([]byte, error) {
	if len(blob) < 3+aes.BlockSize || !bytes.Equal(blob[:3], []byte("v10")) || (len(blob)-3)%aes.BlockSize != 0 || len(password) == 0 {
		return nil, errors.New("credentials: unsupported or invalid macOS safeStorage payload")
	}
	key := pbkdf2.Key(password, []byte("saltysalt"), 1003, 16, sha1.New)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(blob)-3)
	cipher.NewCBCDecrypter(block, bytes.Repeat([]byte{' '}, aes.BlockSize)).CryptBlocks(plain, blob[3:])
	pad := int(plain[len(plain)-1])
	valid := pad > 0 && pad <= aes.BlockSize && pad <= len(plain)
	if valid {
		for _, b := range plain[len(plain)-pad:] {
			if int(b) != pad {
				valid = false
			}
		}
	}
	if !valid {
		clear(plain)
		return nil, errors.New("credentials: macOS safeStorage decryption failed")
	}
	return plain[:len(plain)-pad], nil
}

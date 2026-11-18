package credentials

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

// Electron v44.3.0 restores this synchronous Chromium format in
// patches/chromium/revert_oscrypt_remove_sync_backend.patch. Both shells must
// write this same format to the shared file. Unknown prefixes never become plaintext.
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

func sealLegacyMac(plaintext, password []byte) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("credentials: macOS safeStorage password unavailable")
	}
	key := pbkdf2.Key(password, []byte("saltysalt"), 1003, 16, sha1.New)
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	pad := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := make([]byte, len(plaintext)+pad)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(pad)
	}
	defer clear(padded)
	out := make([]byte, 3+len(padded))
	copy(out, "v10")
	cipher.NewCBCEncrypter(block, bytes.Repeat([]byte{' '}, aes.BlockSize)).CryptBlocks(out[3:], padded)
	return out, nil
}

const (
	macSafeStorageService = "Whale Isle Safe Storage"
	macSafeStorageAccount = "Whale Isle"
)

// getMacSafeStoragePassword reads only the original application's identity.
// Creation is allowed for a new/plaintext profile or a successfully decoded
// MyGo-format migration, never to replace a missing key for an existing CBC file.
func getMacSafeStoragePassword(create bool, read func() ([]byte, error), add func([]byte) error) ([]byte, error) {
	password, err := read()
	if errors.Is(err, errKeychainNotFound) && create {
		clear(password)
		random := make([]byte, 16)
		if _, err = rand.Read(random); err != nil {
			clear(random)
			return nil, err
		}
		password = make([]byte, base64.StdEncoding.EncodedLen(len(random)))
		base64.StdEncoding.Encode(password, random)
		clear(random)
		err = add(password)
		if errors.Is(err, errKeychainDuplicate) {
			clear(password)
			password, err = read()
		}
	}
	if err != nil {
		clear(password)
		return nil, err
	}
	if len(password) == 0 || len(password) > 4096 {
		clear(password)
		return nil, errors.New("credentials: macOS safeStorage password has an invalid length")
	}
	return password, nil
}

package credentials

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
)

func TestLegacyMacSafeStorageSynthetic(t *testing.T) {
	// Independently generated with Python hashlib.pbkdf2_hmac + OpenSSL AES-CBC.
	// Neither the password nor JSON came from an account or a Keychain.
	blob, _ := hex.DecodeString("7631305b4e217df97c3793d529753da8a42be0f19c7913ee062272e72b8667f4f24b0a")
	before := bytes.Clone(blob)
	got, err := openLegacyMac(blob, []byte("synthetic-whale-fixture"))
	if err != nil || string(got) != `{"apiKey":"synthetic-only"}` {
		t.Fatalf("known-answer mismatch: %v", err)
	}
	if !bytes.Equal(blob, before) {
		t.Fatal("decoder mutated envelope")
	}
	if _, err := openLegacyMac(blob, []byte("wrong-fixture")); err == nil {
		t.Fatal("wrong password accepted")
	}
	for _, bad := range [][]byte{nil, []byte("plaintext"), blob[:3], blob[:len(blob)-1], append([]byte("v11"), blob[3:]...), append([]byte("v20"), blob[3:]...)} {
		if _, err := openLegacyMac(bad, []byte("synthetic-whale-fixture")); err == nil {
			t.Fatal("unknown/truncated format accepted")
		}
	}
	if _, err := openLegacyMac(blob, nil); err == nil {
		t.Fatal("missing password accepted")
	}
}

func TestMacSafeStorageEncoderMatchesIndependentCiphertext(t *testing.T) {
	// This independent known-answer vector is shared with the decoder fixture.
	want, _ := hex.DecodeString("7631305b4e217df97c3793d529753da8a42be0f19c7913ee062272e72b8667f4f24b0a")
	password := []byte("synthetic-whale-fixture")
	plain := []byte(`{"apiKey":"synthetic-only"}`)
	got, err := sealLegacyMac(plain, password)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("Electron CBC wire compatibility: %v", err)
	}
	if string(plain) != `{"apiKey":"synthetic-only"}` || string(password) != "synthetic-whale-fixture" {
		t.Fatal("encoder changed caller input")
	}
	for _, plain := range [][]byte{nil, []byte("1234567890123456"), []byte("中文 🔑 credential")} {
		ciphertext, err := sealLegacyMac(plain, password)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := openLegacyMac(ciphertext, password)
		if err != nil || !bytes.Equal(roundTrip, plain) {
			t.Fatalf("padding or UTF-8 changed: %v", err)
		}
	}
	if _, err := sealLegacyMac(plain, nil); err == nil {
		t.Fatal("encrypted without a password")
	}
}

func TestMacSafeStoragePasswordUsesExistingIdentityAndCreatesOnlyWhenAllowed(t *testing.T) {
	existing := []byte("original-application-password")
	adds := 0
	add := func([]byte) error { adds++; return nil }
	got, err := getMacSafeStoragePassword(true, func() ([]byte, error) { return bytes.Clone(existing), nil }, add)
	if err != nil || !bytes.Equal(got, existing) || adds != 0 {
		t.Fatal("existing shared identity was replaced")
	}
	clear(got)
	for _, denied := range []error{errKeychainNotFound, errors.New("authorization denied")} {
		for _, create := range []bool{false, true} {
			if create && errors.Is(denied, errKeychainNotFound) {
				continue
			}
			if _, err := getMacSafeStoragePassword(create, func() ([]byte, error) { return nil, denied }, add); !errors.Is(err, denied) || adds != 0 {
				t.Fatal("unreadable existing credentials created a replacement identity")
			}
		}
	}
	var added []byte
	got, err = getMacSafeStoragePassword(true, func() ([]byte, error) { return nil, errKeychainNotFound }, func(password []byte) error {
		added = bytes.Clone(password)
		return nil
	})
	random, decodeErr := base64.StdEncoding.DecodeString(string(added))
	if err != nil || decodeErr != nil || len(random) != 16 || !bytes.Equal(got, added) {
		t.Fatalf("new identity must use Chromium's 128-bit base64 password: %v", err)
	}
}

func TestMacSafeStoragePasswordCreationRaceReadsWinner(t *testing.T) {
	winner := []byte("existing-concurrent-electron-key")
	var loser []byte
	reads := 0
	got, err := getMacSafeStoragePassword(true, func() ([]byte, error) {
		reads++
		if reads == 1 {
			return nil, errKeychainNotFound
		}
		return bytes.Clone(winner), nil
	}, func(password []byte) error {
		loser = password
		return errKeychainDuplicate
	})
	if err != nil || !bytes.Equal(got, winner) || reads != 2 {
		t.Fatalf("duplicate creation did not retain the winning identity: %v", err)
	}
	if !bytes.Equal(loser, make([]byte, len(loser))) {
		t.Fatal("discarded password was not cleared")
	}
}

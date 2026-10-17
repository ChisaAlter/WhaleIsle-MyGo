package credentials

import (
	"bytes"
	"encoding/hex"
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

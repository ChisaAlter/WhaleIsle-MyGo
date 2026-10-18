package credentials

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type memoryKeychain struct {
	key  []byte
	err  error
	adds int
}

func (m *memoryKeychain) get(string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.key == nil {
		return nil, errKeychainNotFound
	}
	return m.key, nil
}
func (m *memoryKeychain) add(_ string, k []byte) error {
	m.adds++
	m.key = append([]byte(nil), k...)
	return nil
}

// This fixture detects plaintext fallback, key creation during reads, and
// replacing existing credential bytes after a denied Keychain access.
func TestKeychainCredentialPersistence(t *testing.T) {
	old := credentialOS
	t.Cleanup(func() { credentialOS = old })
	backend := &memoryKeychain{key: bytes.Repeat([]byte{7}, 32)}
	credentialOS = credentialProvider{version: KeychainEnvelopeVersion, backend: "macOS Keychain + AES-256-GCM", supported: true, key: func(dir string, create bool) ([]byte, error) { return keychainKey(backend, dir, create) }}
	dir := t.TempDir()
	want := sampleCreds()
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(before, []byte(want.APIKey)) || !bytes.Contains(before, []byte(KeychainEnvelopeVersion)) {
		t.Fatal("incorrect storage envelope")
	}
	got, err := Load(dir)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip: %v", err)
	}
	backend.err = errors.New("keychain locked")
	if err := Save(dir, want); err == nil {
		t.Fatal("denied save accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("denied save changed file")
	}
	if got, err := Load(dir); err == nil || !reflect.DeepEqual(got, Credentials{}) {
		t.Fatal("denied read exposed credentials")
	}
	if s := InspectStorage(dir); s.Mode != "encrypted" || !s.SupportedPlatform {
		t.Fatalf("metadata: %+v", s)
	}
	backend.err = nil
	backend.key = nil
	if _, err := Load(dir); !errors.Is(err, errKeychainNotFound) {
		t.Fatalf("missing key: %v", err)
	}
	if err := Save(dir, want); !errors.Is(err, errKeychainNotFound) {
		t.Fatalf("missing key save: %v", err)
	}
	after, _ = os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("missing key save replaced file")
	}
	if backend.adds != 0 {
		t.Fatal("read created a key")
	}
}

package credentials

import (
	"bytes"
	"errors"
	"testing"
)

type racingKeychain struct {
	reads, adds int
	generated   []byte
	winner      []byte
}

func (r *racingKeychain) get(string) ([]byte, error) {
	r.reads++
	if r.reads == 1 {
		return nil, errKeychainNotFound
	}
	return bytes.Clone(r.winner), nil
}
func (r *racingKeychain) add(_ string, key []byte) error {
	r.adds++
	r.generated = key // observe ownership after duplicate status is returned
	return errKeychainDuplicate
}

func TestKeychainDuplicateCreationRereadsAndWipesLoser(t *testing.T) {
	backend := &racingKeychain{winner: bytes.Repeat([]byte{9}, 32)}
	key, err := keychainKey(backend, t.TempDir(), true)
	if err != nil || !bytes.Equal(key, backend.winner) || backend.reads != 2 || backend.adds != 1 {
		t.Fatalf("creation race: %v", err)
	}
	if !bytes.Equal(backend.generated, make([]byte, 32)) {
		t.Fatal("losing generated key was not cleared")
	}
	clear(key)
	if !bytes.Equal(backend.winner, bytes.Repeat([]byte{9}, 32)) {
		t.Fatal("caller owned backend storage")
	}
}

func TestKeychainMalformedKeyIsCleared(t *testing.T) {
	bad := []byte{1, 2, 3}
	backend := &returnedKeychain{key: bad}
	if _, err := keychainKey(backend, t.TempDir(), false); err == nil || !bytes.Equal(bad, make([]byte, len(bad))) {
		t.Fatal("invalid Keychain key retained")
	}
}

type returnedKeychain struct{ key []byte }

func (r *returnedKeychain) get(string) ([]byte, error) { return r.key, nil }
func (*returnedKeychain) add(string, []byte) error     { return errors.New("unexpected create") }

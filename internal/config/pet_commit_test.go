package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A growth refresh that settles nothing new commits nothing: it neither
// rewrites config.json nor collides with another writer's edit.
func TestPetCommitSkipsUnchangedPet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"port":4000,"live2dPet":{"enabled":true,"growth":{"points":1}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(1_790_000_000_000)
	samples := map[string]int64{strings.Repeat("a", 64): 10}
	if err := s.RefreshPetGrowth(samples, now); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	external := append(append([]byte(nil), written...), '\n')
	if err := os.WriteFile(path, external, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshPetGrowth(samples, now); err != nil {
		t.Fatalf("an unchanged refresh tried to write: %v", err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, external) {
		t.Fatal("an unchanged refresh rewrote config.json")
	}
}

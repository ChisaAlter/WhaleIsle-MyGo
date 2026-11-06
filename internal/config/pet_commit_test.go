package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/petgrowth"
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

func TestPetCareAfterWorkspaceMergeRetainsFreshGrowth(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(1_790_000_000_000)
	id := strings.Repeat("a", 64)
	usage := petgrowth.Usage{Source: "magpie", Samples: map[string]int64{id: 100}}
	if err := s.RefreshPetGrowthUsage(usage, now); err != nil {
		t.Fatal(err)
	}
	external, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	usage.Samples[id] = 120
	if err := external.RefreshPetGrowthUsage(usage, now); err != nil {
		t.Fatal(err)
	}
	want, err := external.PetGrowth(now)
	if err != nil || want.Available != 20 {
		t.Fatalf("fresh growth: %+v, %v", want, err)
	}
	if _, err := s.MergeWorkspaceNavigationPreferences("host", WorkspaceNavigationPreferences{}, WorkspaceNavigationPreferences{GroupBy: "flat"}); err != nil {
		t.Fatal(err)
	}
	if err := s.PatPet(now); err != nil {
		t.Fatal(err)
	}
	got, err := s.PetGrowth(now)
	if err != nil || got.Available != want.Available {
		t.Fatalf("care revived stale growth: %+v, want %+v, err %v", got, want, err)
	}
	if err := s.RefreshPetGrowthUsage(usage, now); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err = reopened.PetGrowth(now)
	if err != nil || got.Available != want.Available {
		t.Fatalf("later refresh changed settled food: %+v, want %+v, err %v", got, want, err)
	}
}

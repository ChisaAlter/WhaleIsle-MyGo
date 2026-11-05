package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/petgrowth"
)

func TestPetGrowthSourceMigrationPersistsThenFeedsOnce(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("a", 64)
	raw := `{"live2dPet":{"growth":{"points":3,"tokensFed":7,"tokensSeen":50,"baseline":0,"food":{"version":1,"available":43,"future":"keep","seen":{"` + id + `":50}}}}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.Local)
	usage := petgrowth.Usage{Source: "magpie", Samples: map[string]int64{id: 100}}
	if err := s.RefreshPetGrowthUsage(usage, now); err != nil {
		t.Fatal(err)
	}
	usage.Samples[id] = 120
	if err := s.RefreshPetGrowthUsage(usage, now); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	v, err := reopened.PetGrowth(now)
	if err != nil || v.Points != 3 || v.Available != 20 || v.TokensFed != 0 {
		t.Fatalf("reopened migration %+v %v", v, err)
	}
	if fed, err := reopened.FeedPetTokens(now); err != nil || fed != 20 {
		t.Fatalf("explicit feed %d %v", fed, err)
	}
	if fed, err := reopened.FeedPetTokens(now); err != nil || fed != 0 {
		t.Fatalf("duplicate feed %d %v", fed, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil || !json.Valid(data) || !strings.Contains(string(data), `"source": "magpie"`) || !strings.Contains(string(data), `"future": "keep"`) || !strings.Contains(string(data), `"accountingVersion": 2`) {
		t.Fatalf("source or unknown fields lost: %v %s", err, data)
	}
}

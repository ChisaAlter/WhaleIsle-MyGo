package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateProtectionStagedHandoff(t *testing.T) {
	for _, mode := range []string{"success", "changed", "launch-failed", "already-started"} {
		t.Run(mode, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "fixture.exe")
			data := []byte("not an executable; fixture only")
			if err := os.WriteFile(p, data, 0600); err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(data)
			staged := ApplyResult{Path: p, SHA256: hex.EncodeToString(hash[:]), Size: int64(len(data))}
			if mode == "changed" {
				if err := os.WriteFile(p, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "already-started" {
				staged.Started = true
			}
			calls := 0
			out, err := startStaged(staged, func(path string) error {
				calls++
				if path != p {
					t.Fatal("wrong installer")
				}
				if mode == "launch-failed" {
					return errors.New("fixture failure")
				}
				return nil
			})
			if (err == nil) != (mode == "success") {
				t.Fatalf("unexpected error: %v", err)
			}
			if (calls == 1) != (mode == "success" || mode == "launch-failed") {
				t.Fatalf("launch calls=%d", calls)
			}
			if mode != "already-started" && out.Started != (mode == "success") {
				t.Fatal("false installer start result")
			}
		})
	}
}

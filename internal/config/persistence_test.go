package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readConfig(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestUpdateRejectsExternalEditBeforeCallback checks that an edit made after
// Open is detected before the callback runs, so a conflicting update has no
// side effects and the external bytes survive.
func TestUpdateRejectsExternalEditBeforeCallback(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"port":4000}`)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	external := `{"port":9999,"note":"edited elsewhere"}`
	writeConfig(t, dir, external)

	invoked := false
	err = s.Update(func(c *Config) { invoked = true; c.Port = 5000 })
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if invoked {
		t.Fatal("callback ran for a conflicting update")
	}
	if got := readConfig(t, dir); got != external {
		t.Fatalf("external edit overwritten: %s", got)
	}
	if s.Get().Port != 4000 {
		t.Fatalf("store state changed by a rejected update: %+v", s.Get())
	}
}

// TestUpdateDetectsFileAppearingAndDisappearing covers both baseline
// existence transitions: a file that was absent appears, and one that was
// present disappears.
func TestUpdateDetectsFileAppearingAndDisappearing(t *testing.T) {
	t.Run("appears", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Open(dir) // file absent
		if err != nil {
			t.Fatal(err)
		}
		writeConfig(t, dir, `{"port":4000}`)
		if err := s.Update(func(c *Config) { c.Port = 5000 }); !errors.Is(err, ErrConflict) {
			t.Fatalf("want ErrConflict, got %v", err)
		}
		if got := readConfig(t, dir); got != `{"port":4000}` {
			t.Fatalf("external file overwritten: %s", got)
		}
	})

	t.Run("disappears", func(t *testing.T) {
		dir := t.TempDir()
		writeConfig(t, dir, `{"port":4000}`)
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(dir, FileName)); err != nil {
			t.Fatal(err)
		}
		if err := s.Update(func(c *Config) { c.Port = 5000 }); !errors.Is(err, ErrConflict) {
			t.Fatalf("want ErrConflict, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, FileName)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("rejected update recreated the file: %v", err)
		}
	})
}

// TestUpdateDetectsByteLevelEdits checks that the comparison is byte-based:
// even an edit that reparses to the same values is a conflict.
func TestUpdateDetectsByteLevelEdits(t *testing.T) {
	for name, external := range map[string]string{
		"whitespace only": "{\n  \"port\": 4000\n}\n",
		"malformed":       `{"port": 4000`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, `{"port":4000}`)
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			writeConfig(t, dir, external)
			if err := s.Update(func(c *Config) { c.Port = 5000 }); !errors.Is(err, ErrConflict) {
				t.Fatalf("want ErrConflict, got %v", err)
			}
			if got := readConfig(t, dir); got != external {
				t.Fatalf("external edit overwritten: %s", got)
			}
		})
	}
}

// TestUpdateDetectsEditDuringCallback checks the second comparison, taken
// immediately before the atomic replace: an edit made while the callback ran
// must still be rejected rather than overwritten.
func TestUpdateDetectsEditDuringCallback(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	external := `{"port":9999,"note":"edited during callback"}`
	err = s.Update(func(c *Config) {
		c.Port = 5000
		writeConfig(t, dir, external)
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if got := readConfig(t, dir); got != external {
		t.Fatalf("concurrent edit overwritten: %s", got)
	}
	if s.Get().Port != Defaults().Port {
		t.Fatalf("store state changed by a rejected update: %+v", s.Get())
	}
}

// TestTwoStaleStoresSequentialUpdate checks that the first store wins and the
// second, still holding the pre-write baseline, is refused.
func TestTwoStaleStoresSequentialUpdate(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := first.Update(func(c *Config) { c.Port = 5000 }); err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := second.Update(func(c *Config) { c.Port = 6000 }); !errors.Is(err, ErrConflict) {
		t.Fatalf("second update: want ErrConflict, got %v", err)
	}
	if got := first.Get().Port; got != 5000 {
		t.Fatalf("first store lost its committed value: %d", got)
	}
}

// TestConcurrentUpdatesExactlyOneWins starts two stores from the same absent
// baseline and commits concurrently; the package-local commit mutex must
// serialize the compare-and-replace so exactly one succeeds.
func TestConcurrentUpdatesExactlyOneWins(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, s := range []*Store{first, second} {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			<-start
			results <- s.Update(func(c *Config) { c.Port = 5000 })
		}(s)
	}
	close(start)
	wg.Wait()
	close(results)

	var ok, conflicts int
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Fatalf("want exactly one success and one conflict, got %d ok / %d conflict", ok, conflicts)
	}
	if got := readConfig(t, dir); got == "" {
		t.Fatal("winner did not write the file")
	}
}

// TestRepeatedUpdatesKeepUnknownKeys checks that a long run of ordinary
// updates never triggers a false conflict and never drops carried-through
// keys.
func TestRepeatedUpdatesKeepUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"pet":{"xRatio":0.9},"pluginRecovery":{"skipUserPlugins":true,"reason":"manual"}}`)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		if err := s.Update(func(c *Config) { c.Port = 4000 + i }); err != nil {
			t.Fatalf("update %d: %v", i, err)
		}
	}
	got := readConfig(t, dir)
	var raw map[string]any
	if err := json.Unmarshal([]byte(got), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["port"] != float64(4005) {
		t.Fatalf("last write not persisted: %s", got)
	}
	if pet, ok := raw["pet"].(map[string]any); !ok || pet["xRatio"] != 0.9 {
		t.Fatalf("pet key lost across updates: %v", raw["pet"])
	}
	if rec, ok := raw["pluginRecovery"].(map[string]any); !ok || rec["reason"] != "manual" {
		t.Fatalf("pluginRecovery key lost across updates: %v", raw["pluginRecovery"])
	}
	if s.Get().Port != 4005 {
		t.Fatalf("in-memory value drifted: %+v", s.Get())
	}
}

// TestFailedWriteKeepsBaselineAndCleansTemp forces the rename to fail and
// checks the baseline is untouched, so a later real update still succeeds and
// no temp file is left behind.
func TestFailedWriteKeepsBaselineAndCleansTemp(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"port":4000}`)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.rename = func(oldpath, newpath string) error { return errors.New("injected rename failure") }
	if err := s.Update(func(c *Config) { c.Port = 5000 }); err == nil {
		t.Fatal("Update succeeded despite an injected rename failure")
	}
	if got := readConfig(t, dir); got != `{"port":4000}` {
		t.Fatalf("file changed after a failed write: %s", got)
	}
	if s.Get().Port != 4000 {
		t.Fatalf("memory changed after a failed write: %+v", s.Get())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != FileName {
		t.Fatalf("temp file was not cleaned up: %v", entries)
	}

	// The baseline must still match, so the next (real) update goes through.
	s.rename = nil
	if err := s.Update(func(c *Config) { c.Port = 5000 }); err != nil {
		t.Fatalf("update after a failed write: %v", err)
	}
	if s.Get().Port != 5000 {
		t.Fatalf("update after a failed write did not commit: %+v", s.Get())
	}
}

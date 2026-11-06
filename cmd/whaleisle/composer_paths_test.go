package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitComposerPaths(t *testing.T, s *mainState, root string) []string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		paths := s.composerPathsLocked(root)
		s.mu.Unlock()
		if paths != nil {
			return paths
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("index never became available")
	return nil
}

// Building the @ index inside the view build froze the window for as long as the
// walk took. The first request must return at once with nothing, and the files
// arrive from the worker.
func TestComposerPathsIndexesOffTheCallingThread(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(root))
	for _, rel := range []string{"a.txt", "dir/b.txt"} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := &mainState{}
	s.mu.Lock()
	first := s.composerPathsLocked(root)
	loading := s.composerPathsLoading
	s.mu.Unlock()
	if first != nil || loading != root {
		t.Fatalf("first request must start a worker and return nothing: paths=%v loading=%q", first, loading)
	}
	got := waitComposerPaths(t, s, root)
	want := map[string]bool{"a.txt": true, "dir": true, "dir/b.txt": true}
	if len(got) != len(want) {
		t.Fatalf("paths = %v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected path %q in %v", p, got)
		}
	}
	s.mu.Lock()
	again := s.composerPathsLocked(root)
	idle := s.composerPathsLoading == ""
	s.mu.Unlock()
	if len(again) != len(got) || !idle {
		t.Fatal("a finished index must be reused without another walk")
	}
}

// Files of the previous workspace must not be offered while the next one loads.
func TestComposerPathsDropsThePreviousRootWhileLoading(t *testing.T) {
	one, two := t.TempDir(), t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(one))
	if err := os.WriteFile(filepath.Join(one, "old.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(two, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &mainState{}
	if got := waitComposerPaths(t, s, one); len(got) != 1 || got[0] != "old.txt" {
		t.Fatalf("first root: %v", got)
	}
	s.mu.Lock()
	during := s.composerPathsLocked(two)
	stale := s.composerPaths
	s.mu.Unlock()
	if during != nil || stale != nil {
		t.Fatalf("stale paths offered for the new root: %v / %v", during, stale)
	}
	if got := waitComposerPaths(t, s, two); len(got) != 1 || got[0] != "new.txt" {
		t.Fatalf("second root: %v", got)
	}
}

// A root that cannot be read is retried after a delay, not on every frame.
func TestComposerPathsBacksOffAfterAFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	s := &mainState{}
	s.mu.Lock()
	s.composerPathsLocked(missing)
	s.mu.Unlock()
	deadline := time.Now().Add(10 * time.Second)
	for {
		s.mu.Lock()
		failed := s.composerPathsFailedRoot == missing && s.composerPathsLoading == ""
		s.mu.Unlock()
		if failed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("failure was not recorded")
		}
		time.Sleep(5 * time.Millisecond)
	}
	s.mu.Lock()
	got := s.composerPathsLocked(missing)
	started := s.composerPathsLoading != ""
	s.mu.Unlock()
	if got != nil || started {
		t.Fatal("a failed root was walked again immediately")
	}
}

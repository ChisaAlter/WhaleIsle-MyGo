package recovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRestartAndConflictingInstance(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Update(func(s *State) {
		s.Workspace = "/project"
		s.SelectedSession = "session-1"
		s.Layout = Layout{Sidebar: 300, Right: 450, RightOpen: true}
		s.Drafts = map[string]json.RawMessage{"session-1": json.RawMessage(`{"text":"unfinished draft"}`)}
		s.Files = json.RawMessage(`{"dirty":"unsaved buffer"}`)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Update(func(s *State) { s.Workspace = "stale" }); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale writer: %v", err)
	}
	resumed, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := resumed.Snapshot()
	if got.Workspace != "/project" || got.SelectedSession != "session-1" || got.Layout.Sidebar != 300 || !bytes.Contains(got.Drafts["session-1"], []byte("unfinished")) || !bytes.Contains(got.Files, []byte("unsaved")) {
		t.Fatalf("resume lost state: %+v", got)
	}
	got.Drafts["session-1"][0] = 'x'
	if !json.Valid(resumed.Snapshot().Drafts["session-1"]) {
		t.Fatal("snapshot aliased")
	}
}

func TestInvalidAndLockedFilesArePreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	broken := []byte(`{"drafts": incomplete`)
	if err := os.WriteFile(path, broken, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("corrupt file accepted")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, broken) {
		t.Fatal("corrupt original changed")
	}
	os.Remove(path)
	s, _ := Open(dir)
	if err := os.WriteFile(path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(v *State) { v.Workspace = "blocked" }); !errors.Is(err, ErrBusy) {
		t.Fatalf("locked writer: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("locked write changed disk")
	}
	if s.Snapshot().Workspace != "" {
		t.Fatal("failed write changed memory")
	}
}

package assistant

import (
	"os"
	"testing"
)

// blockStoreRoot replaces the persistence root with a regular file so every
// write under it fails.
func blockStoreRoot(t *testing.T, root string) {
	t.Helper()
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func unblockStoreRoot(t *testing.T, root string) {
	t.Helper()
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestAppendTurnReportsHistorySaveFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := s.Root()
	blockStoreRoot(t, root)

	s.AppendTurn(Turn{Prompt: "prompt", Response: "answer"})
	snap := s.Snapshot().Value
	if snap.LastErr == nil {
		t.Fatal("history save failure was not reported in LastErr")
	}
	if len(snap.History) != 1 {
		t.Fatalf("in-memory history = %d turns, want 1", len(snap.History))
	}

	unblockStoreRoot(t, root)
	s.AppendTurn(Turn{Prompt: "second", Response: "answer"})
	if got := s.Snapshot().Value.LastErr; got != nil {
		t.Fatalf("stale save failure still shown after a successful write: %v", got)
	}
}

func TestSetModeReportsModeSaveFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := s.Root()
	blockStoreRoot(t, root)

	s.SetMode(ModeDshbot)
	snap := s.Snapshot().Value
	if snap.Mode != ModeDshbot {
		t.Fatalf("mode = %q, want %q", snap.Mode, ModeDshbot)
	}
	if snap.LastErr == nil {
		t.Fatal("mode save failure was not reported in LastErr")
	}

	unblockStoreRoot(t, root)
	s.SetMode(ModeWhale)
	if got := s.Snapshot().Value.LastErr; got != nil {
		t.Fatalf("stale save failure still shown after a successful write: %v", got)
	}
}

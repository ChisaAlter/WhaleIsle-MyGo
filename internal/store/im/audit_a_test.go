package im

import (
	"os"
	"testing"
)

func TestFinishRefreshReportsChannelSaveFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := s.root
	s.mu.Lock()
	s.rows = nil
	s.mu.Unlock()
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	s.finishRefresh(finishArgs{})
	if got := s.Snapshot().Err; got == "" {
		t.Fatal("default-channel save failure was not surfaced in the view error")
	}
}

func TestFinishRefreshReportsHistorySaveFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := s.root
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	s.finishRefresh(finishArgs{remoteMsgs: []Message{{ID: "m1"}}, channelID: "chan-1"})
	if got := s.Snapshot().Err; got == "" {
		t.Fatal("history save failure was not surfaced in the view error")
	}
}

package im

import (
	"context"
	"os"
	"testing"
)

// grantLeases gives a never-attached store live conn/refresh leases so a
// finishRefresh call is accepted the way an in-flight Refresh would be.
func grantLeases(s *Store) finishArgs {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connLease = s.conn.Renew(context.Background())
	s.refreshLease = s.refresh.Renew(context.Background())
	return finishArgs{client: s.client, plugin: s.plugin, connLease: s.connLease, refreshLease: s.refreshLease}
}

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

	s.finishRefresh(grantLeases(s))
	if got := s.Snapshot().Value.Err; got == nil {
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

	fa := grantLeases(s)
	fa.remoteMsgs = []Message{{ID: "m1"}}
	fa.channelID = "chan-1"
	s.finishRefresh(fa)
	if got := s.Snapshot().Value.Err; got == nil {
		t.Fatal("history save failure was not surfaced in the view error")
	}
}

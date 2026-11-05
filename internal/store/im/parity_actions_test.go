package im

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/dsh"
)

func TestParityRefreshOwnershipAndAuthoritativeEmpty(t *testing.T) {
	s := New()
	old := &dsh.Client{}
	s.Attach(old, nil)
	s.rows = []Entry{{ID: "feishu:old", BotID: "old"}}
	s.selectedID = "feishu:old"
	epoch, generation := s.connectionEpoch, s.refreshGeneration
	s.Detach()
	s.Attach(&dsh.Client{}, nil)
	s.finishRefresh(finishArgs{client: old, epoch: epoch, generation: generation, replaceRows: true, rows: []Entry{{ID: "stale"}}, hasRPC: true, bridgeMode: "cordis"})
	if snap := s.Snapshot(); snap.HasIMRPC || snap.BridgeMode == "cordis" || snap.Rows[0].ID == "stale" {
		t.Fatal("old Host refresh republished facts", snap)
	}
	s.finishRefresh(finishArgs{client: s.client, epoch: s.connectionEpoch, generation: s.refreshGeneration, replaceRows: true, rows: []Entry{}, hasRPC: true, bridgeMode: "cordis"})
	if snap := s.Snapshot(); len(snap.Rows) != 0 || snap.SelectedID != "" || !snap.HasIMRPC {
		t.Fatal("authoritative empty roster retained deleted Bot", snap)
	}
}

func TestParityLocalSaveFailureRetainsDraft(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.root = file
	s.selectedID = "local"
	s.SetDraft("local", "retain me")
	s.SendDraft(context.Background(), "local", "retain me")
	snap := s.Snapshot()
	if snap.Draft != "retain me" || snap.Err == "" || len(snap.Messages) != 0 {
		t.Fatal("failed local persistence cleared or claimed the draft", snap)
	}
}

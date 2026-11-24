package dshbot

import (
	"sync"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
)

func describeResult() pluginrpc.DshbotDescribeResult {
	return pluginrpc.DshbotDescribeResult{
		View: pluginrpc.DshbotSettingsView{
			Value: pluginrpc.DshbotCatalogValue{
				Items: []pluginrpc.DshbotCatalogItem{
					{ID: "b1", Kind: "bot", Name: "Alpha", SessionID: "s1"},
				},
			},
		},
	}
}

func TestRosterSnapshotSharedWhenUnchanged(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := s.Snapshot()
	if b := s.Snapshot(); a != b {
		t.Fatal("unchanged Snapshot must return the same published version")
	}
	allocs := testing.AllocsPerRun(100, func() { _ = s.Snapshot() })
	if allocs != 0 {
		t.Fatalf("unchanged Snapshot allocs = %v", allocs)
	}
}

func TestRosterWritePublishesNewRevision(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	s.SetFromDescribe(describeResult(), false)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("rev %d -> %d", before.Rev, after.Rev)
	}
	if len(after.Value.Rows) != 1 {
		t.Fatalf("rows = %d", len(after.Value.Rows))
	}
	// Mutating the earlier version must not corrupt stored state.
	before.Value.Rows = append(before.Value.Rows, Row{ID: "x"})
	if len(s.Snapshot().Value.Rows) != 1 {
		t.Fatal("mutating a published version leaked into the store")
	}
}

func TestRosterConcurrentReadWrite(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.SetFromDescribe(describeResult(), false)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = s.Snapshot().Value.Rows
			}
		}()
	}
	wg.Wait()
	if len(s.Snapshot().Value.Rows) != 1 {
		t.Fatal("roster corrupted under concurrency")
	}
}

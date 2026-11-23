package im

import (
	"context"
	"sync"
	"testing"
)

func TestSnapshotSharedWhenUnchanged(t *testing.T) {
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

func TestWritePublishesNewRevision(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	if len(before.Value.Rows) == 0 {
		t.Fatal("seeded channels expected")
	}
	if len(before.Value.Rows) < 2 {
		t.Fatal("need two seeded channels")
	}
	other := before.Value.Rows[1]
	s.Select(channelKey(other))
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("rev %d -> %d", before.Rev, after.Rev)
	}
	// SetDraft publishes only on an actual change.
	s.SetDraft(after.Value.SelectedID, after.Value.Draft)
	if s.Snapshot().Rev != after.Rev {
		t.Fatal("unchanged SetDraft must not publish")
	}
	s.SetDraft(after.Value.SelectedID, "typed")
	if got := s.Snapshot(); got.Rev != after.Rev+1 || got.Value.Draft != "typed" {
		t.Fatalf("draft publish rev=%d draft=%q", got.Rev, got.Value.Draft)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.SetDraft("general", "draft")
				s.Select("general")
				s.Refresh(context.Background())
			}
		}(i)
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
}

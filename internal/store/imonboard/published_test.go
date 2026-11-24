package imonboard

import (
	"sync"
	"testing"
)

func TestImonboardSnapshotSharedUntilWrite(t *testing.T) {
	s := &Store{}
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
}

func TestImonboardWritePublishesAndSignals(t *testing.T) {
	s := &Store{}
	before := s.Snapshot()
	changed := s.Changed()
	s.Bind(nil, 1, nil)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("Bind did not publish Rev+1: %p/%d -> %p/%d", before, before.Rev, after, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	if after.Value.Scope == 0 {
		t.Fatal("scope lease did not mint a Scope generation")
	}
}

func TestImonboardHideDropsStaleScope(t *testing.T) {
	s := &Store{}
	s.Bind(nil, 1, nil)
	scope := s.Snapshot().Value.Scope
	s.Hide()
	v := s.Snapshot().Value
	if v.Scope == scope {
		t.Fatal("Hide did not renew the scope lease")
	}
	s.Edit(scope, Draft{Identity: "stale"})
	if s.Snapshot().Value.Draft.Identity != "" {
		t.Fatal("edit against a retired scope was applied")
	}
}

func TestImonboardConcurrentReadWrite(t *testing.T) {
	s := &Store{}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = s.Snapshot().Value
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			s.Bind(nil, uint64(i+1), nil)
			s.Hide()
		}
	}()
	wg.Wait()
}

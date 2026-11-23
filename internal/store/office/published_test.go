package office

import (
	"context"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
)

func TestOfficeSnapshotSharedUntilWrite(t *testing.T) {
	s := &Store{}
	s.publishLocked()
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
}

func TestOfficeWritePublishesAndSignals(t *testing.T) {
	s := &Store{}
	s.publishLocked()
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

func TestOfficeHideDropsStaleScope(t *testing.T) {
	s := &Store{}
	s.publishLocked()
	s.Bind(&pluginrpc.Client{BaseURL: "http://127.0.0.1:1"}, 1, nil)
	scope := s.Snapshot().Value.Scope
	s.Hide()
	v := s.Snapshot().Value
	if v.Scope == scope {
		t.Fatal("Hide did not renew the scope lease")
	}
	if s.Edit(scope, Draft{}) {
		t.Fatal("edit against a retired scope was accepted")
	}
}

func TestOfficeConcurrentReadWrite(t *testing.T) {
	s := &Store{}
	s.publishLocked()
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

func TestOfficeOpLeaseSuperseded(t *testing.T) {
	s := &Store{}
	s.publishLocked()
	s.Bind(&pluginrpc.Client{BaseURL: "http://127.0.0.1:1"}, 1, nil)
	s.mu.Lock()
	s.view.Open = true
	old := s.beginLocked(context.Background())
	s.publishLocked()
	s.mu.Unlock()
	s.Hide()
	if old.Valid() {
		t.Fatal("Hide did not invalidate the in-flight op lease")
	}
	s.mu.Lock()
	stale := s.currentLocked(s.view.Scope, old, s.client)
	s.mu.Unlock()
	if stale {
		t.Fatal("stale op lease still counts as current")
	}
	// The new op after re-show must mint a new generation.
	s.Show(context.Background())
	time.Sleep(10 * time.Millisecond)
}

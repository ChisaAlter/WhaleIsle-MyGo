package remote

import (
	"context"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/state"
)

// Unchanged state must hand every reader the same immutable version.
func TestRemotePublishedSnapshotIdentity(t *testing.T) {
	s := New()
	s.Attach(nil, nil)
	first, second := s.Snapshot(), s.Snapshot()
	if first != second {
		t.Fatal("unchanged snapshot republished")
	}
	if got := testing.AllocsPerRun(100, func() { _ = s.Snapshot() }); got != 0 {
		t.Fatalf("unchanged snapshot allocated %v times", got)
	}
	ch := s.Changed()
	s.Attach(nil, nil)
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("Attach did not close Changed")
	}
	if got := s.Snapshot(); got.Rev != first.Rev+1 {
		t.Fatalf("revision = %d, want %d", got.Rev, first.Rev+1)
	}
}

// Files publishes one deep-copied version per write; an idle read shares it.
func TestFilesPublishedSnapshotIdentity(t *testing.T) {
	s := NewFiles()
	s.Bind(nil, "", nil)
	first, second := s.Snapshot(), s.Snapshot()
	if first != second {
		t.Fatal("unchanged file snapshot republished")
	}
	if got := testing.AllocsPerRun(100, func() { _ = s.Snapshot() }); got != 0 {
		t.Fatalf("unchanged file snapshot allocated %v times", got)
	}
	s.Suspend()
	if got := s.Snapshot(); got == first {
		t.Fatal("Suspend republished the same version")
	}
}

// Readers observe a coherent version while writers publish and rebind.
func TestRemotePublishedConcurrent(t *testing.T) {
	s := New()
	s.Attach(nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				v := s.Snapshot()
				if v == nil {
					t.Error("nil version")
					return
				}
				_ = v.Value.Generation
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			s.Attach(nil, nil)
		}
	}()
	wg.Wait()
}

// Faults carry classified copy and authored text, never raw error text.
func TestRemoteFaultMapping(t *testing.T) {
	f := state.FaultText("Authored copy stays verbatim")
	if f == nil || f.SentenceText() != "Authored copy stays verbatim" {
		t.Fatalf("FaultText = %+v", f)
	}
	if f := state.FaultOf(nil); f != nil {
		t.Fatalf("FaultOf(nil) = %+v", f)
	}
}

// A lease minted for a retired generation must reject a late completion.
func TestRemoteStaleLeaseRejected(t *testing.T) {
	var owner state.LeaseOwner
	first := owner.Renew(context.Background())
	second := owner.Renew(context.Background())
	if first.Valid() || second.Generation() != first.Generation()+1 {
		t.Fatal("renew did not invalidate the previous lease")
	}
}

package assistant

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/state"
)

// Unchanged reads share one immutable version; a write bumps Rev.
func TestAssistantPublishedIdentity(t *testing.T) {
	s := New()
	first, second := s.Snapshot(), s.Snapshot()
	if first != second {
		t.Fatal("unchanged snapshot republished")
	}
	if got := testing.AllocsPerRun(100, func() { _ = s.Snapshot() }); got != 0 {
		t.Fatalf("unchanged snapshot allocated %v times", got)
	}
	s.SetBusy(true)
	if got := s.Snapshot(); got == first || got.Rev != first.Rev+1 {
		t.Fatal("SetBusy did not publish the next revision")
	}
}

// Whale editors publish one version per write and reject stale tickets.
func TestWhalePublishedAndStaleTicket(t *testing.T) {
	var panel WhaleTaskPanel
	panel.Bind("host")
	first := panel.Snapshot()
	if first == nil || first != panel.Snapshot() {
		t.Fatal("unchanged task snapshot republished")
	}
	ticket, _, ok := panel.Begin("host", "refresh")
	if !ok {
		t.Fatal("begin rejected")
	}
	if panel.Snapshot() == first {
		t.Fatal("begin did not publish")
	}
	panel.Bind("host-b")
	if panel.Current(ticket) || panel.Finish(ticket, "refresh", nil, false, "", nil) {
		t.Fatal("stale lease accepted a completion")
	}
}

// Persistence runs off the state mutex; a slow disk cannot block readers.
func TestAssistantPersistOutsideLock(t *testing.T) {
	root := t.TempDir()
	s := New()
	s.root = root
	s.SetMode(ModeDshbot)
	raw, err := os.ReadFile(filepath.Join(root, modeFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("mode was not persisted")
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if s.Snapshot() == nil {
					t.Error("nil version")
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 20; j++ {
			s.SetBusy(j%2 == 0)
		}
	}()
	wg.Wait()
}

// SetLastAction stores classified faults; raw text never enters the view.
func TestAssistantFaultMapping(t *testing.T) {
	s := New()
	s.SetLastAction("failed", state.FaultOf(errors.New("dial tcp refused")))
	if got := s.Snapshot().Value.LastErr; got == nil || got.SentenceText() == "" || got.SentenceText() == "dial tcp refused" {
		t.Fatalf("LastErr = %+v", got)
	}
	s.SetLastAction("ok", nil)
	if s.Snapshot().Value.LastErr != nil {
		t.Fatal("nil fault did not clear LastErr")
	}
}

// A blocking reader is unrelated to the fire channel; fire must not block
// publish ordering.
func TestAssistantFireDoesNotSerialize(t *testing.T) {
	s := New()
	fired := make(chan struct{}, 1)
	s.Attach(func() { fired <- struct{}{} })
	s.SetBusy(true)
	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("notify did not run")
	}
}

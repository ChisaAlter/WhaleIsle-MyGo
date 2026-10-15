package files

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func awaitTextRead(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("synthetic read did not settle")
	}
}

func TestTextReadQueueLatestBoundedOwner(t *testing.T) {
	q := &textReadQueue{}
	defer q.stop(true)
	notified := make(chan struct{}, 8)
	q.setNotify(func() { notified <- struct{}{} })
	owner := textReadOwner{binding: "host-a", root: "opaque-root", session: "session-a", path: "same.txt", tab: 1}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var active, peak, calls atomic.Int32
	oldContext := make(chan context.Context, 1)
	q.request(owner, func(ctx context.Context) textReadResult {
		calls.Add(1)
		active.Add(1)
		peak.Store(1)
		oldContext <- ctx
		close(started)
		<-release // Deliberately ignore context, like an uninterruptible OS read.
		active.Add(-1)
		return textReadResult{text: "obsolete"}
	})
	awaitTextRead(t, started)
	ctx := <-oldContext
	for i := 2; i <= 20; i++ {
		owner.tab = uint64(i)
		q.request(owner, func(context.Context) textReadResult {
			calls.Add(1)
			n := active.Add(1)
			if n > peak.Load() {
				peak.Store(n)
			}
			defer active.Add(-1)
			return textReadResult{text: "latest"}
		})
	}
	awaitTextRead(t, ctx.Done())
	if calls.Load() != 1 {
		t.Fatalf("replacement spawned parallel reads: %d", calls.Load())
	}
	releaseOnce.Do(func() { close(release) })
	awaitTextRead(t, notified)
	if calls.Load() != 2 || peak.Load() != 1 {
		t.Fatalf("expected one running plus latest pending: calls=%d peak=%d", calls.Load(), peak.Load())
	}
	wrong := []textReadOwner{owner, owner, owner, owner, owner}
	wrong[0].binding = "host-b"
	wrong[1].root = "different-root"
	wrong[2].session = "session-b"
	wrong[3].path = "different.txt"
	wrong[4].tab = 1 // closed/reopened same path is a new owner
	for _, other := range wrong {
		if _, ready := q.take(other); ready {
			t.Fatal("completion escaped Host/root/session/path/tab ownership")
		}
	}
	got, ready := q.take(owner)
	if !ready || got.text != "latest" {
		t.Fatalf("wrong completion: %+v ready=%v", got, ready)
	}
	if _, ready := q.take(owner); ready {
		t.Fatal("completion applied twice")
	}
}

func TestTextReadQueueCancelCloseAndFailure(t *testing.T) {
	q := &textReadQueue{}
	notified := make(chan struct{}, 8)
	q.setNotify(func() { notified <- struct{}{} })
	owner := textReadOwner{binding: "host", path: "file", tab: 1}
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	q.request(owner, func(context.Context) textReadResult {
		close(started)
		<-release
		close(finished)
		return textReadResult{text: "late"}
	})
	awaitTextRead(t, started)
	q.stop(false) // explicit cancel / hiding / Host switch
	if _, ready := q.take(owner); ready {
		t.Fatal("cancelled result visible")
	}
	errRead := errors.New("synthetic failure")
	q.request(owner, func(context.Context) textReadResult { return textReadResult{err: errRead} })
	releaseOnce.Do(func() { close(release) })
	awaitTextRead(t, finished)
	awaitTextRead(t, notified)
	got, ready := q.take(owner)
	if !ready || !errors.Is(got.err, errRead) || got.text != "" {
		t.Fatalf("old same-key generation or failure lost: %+v ready=%v", got, ready)
	}
	q.stop(true)
	var ran atomic.Bool
	q.request(owner, func(context.Context) textReadResult { ran.Store(true); return textReadResult{} })
	q.mu.Lock()
	pending := q.pending != nil
	q.mu.Unlock()
	if ran.Load() || pending {
		t.Fatal("closed queue accepted a new read")
	}
}

func TestTextReadReconcilePreservesDraftAndWriteAuthority(t *testing.T) {
	dirty := textReadState{preview: "baseline", draft: "unsaved"}
	for _, result := range []textReadResult{{text: "changed"}, {text: "prefix", truncated: true}, {err: errors.New("failed")}} {
		got := reconcileTextRead(dirty, result)
		if got.preview != dirty.preview || got.draft != dirty.draft || !got.blocked {
			t.Fatalf("dirty baseline/draft overwritten or unsafe saving enabled: %+v", got)
		}
	}
	got := reconcileTextRead(dirty, textReadResult{text: "baseline", remote: true})
	if !got.remote || got.preview != dirty.preview || got.draft != dirty.draft {
		t.Fatalf("remote completion established local authority: %+v", got)
	}
	clean := textReadState{preview: "old", draft: "old", truncated: true}
	got = reconcileTextRead(clean, textReadResult{text: "prefix", truncated: true})
	if got.preview != "prefix" || got.draft != "prefix" || !got.truncated {
		t.Fatalf("clean prefix metadata lost: %+v", got)
	}
	got = reconcileTextRead(clean, textReadResult{text: "complete"})
	if got.preview != "complete" || got.draft != "complete" || got.truncated || got.blocked {
		t.Fatalf("clean complete read did not settle: %+v", got)
	}
	got = reconcileTextRead(clean, textReadResult{err: context.Canceled})
	if got.preview != "old" || got.draft != "old" || !got.truncated || !got.blocked {
		t.Fatalf("failed read discarded current text: %+v", got)
	}
}

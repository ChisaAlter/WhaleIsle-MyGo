package state

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPublishedLoadIsSharedAndVersioned(t *testing.T) {
	var p Published[[]int]
	if p.Load() != nil {
		t.Fatal("Load before Publish must be nil")
	}
	v1 := p.Publish([]int{1, 2, 3})
	if v1.Rev != 1 {
		t.Fatalf("first rev %d", v1.Rev)
	}
	if p.Load() != v1 {
		t.Fatal("Load must return the published pointer")
	}
	v2 := p.Publish([]int{4})
	if v2.Rev != 2 || p.Load() != v2 {
		t.Fatal("Publish must bump Rev and swap the pointer")
	}
	// AllocsPerRun: reading an unchanged publication allocates nothing.
	if n := testing.AllocsPerRun(100, func() { _ = p.Load() }); n != 0 {
		t.Fatalf("Load allocated %v times", n)
	}
}

func TestPublishedChangedClosesOnce(t *testing.T) {
	var p Published[int]
	ch := p.Changed()
	select {
	case <-ch:
		t.Fatal("changed closed before publish")
	default:
	}
	p.Publish(1)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("publish did not close the change channel")
	}
	// A channel taken now must only close on the *next* publish.
	ch2 := p.Changed()
	select {
	case <-ch2:
		t.Fatal("stale close leaked into the next generation")
	default:
	}
	p.Publish(2)
	<-ch2
}

func TestPublishedConcurrent(t *testing.T) {
	var p Published[int]
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				p.Publish(i)
				_ = p.Load()
				_ = p.Changed()
			}
		}(w)
	}
	wg.Wait()
	if last := p.Load(); last == nil || last.Rev == 0 {
		t.Fatal("no version after concurrent publishes")
	}
}

func TestLeaseOwnerGenerations(t *testing.T) {
	var o LeaseOwner
	var zero Lease
	if zero.Valid() || zero.Generation() != 0 {
		t.Fatal("zero lease must be invalid")
	}
	a := o.Renew(context.Background())
	if !a.Valid() || a.Generation() != 1 {
		t.Fatalf("first lease valid=%v gen=%d", a.Valid(), a.Generation())
	}
	b := o.Renew(context.Background())
	if a.Valid() {
		t.Fatal("the superseded lease must be invalid")
	}
	select {
	case <-a.Context().Done():
	default:
		t.Fatal("superseded lease context must be cancelled")
	}
	if !b.Valid() || b.Generation() != 2 {
		t.Fatalf("second lease valid=%v gen=%d", b.Valid(), b.Generation())
	}
	o.Retire()
	if b.Valid() {
		t.Fatal("retire must invalidate the active lease")
	}
	select {
	case <-b.Context().Done():
	default:
		t.Fatal("retire must cancel the lease context")
	}
	c := o.Renew(context.Background())
	if !c.Valid() || c.Generation() != 3 {
		t.Fatal("renew after retire must keep numbering generations")
	}
}

func TestLeaseContextFollowsParent(t *testing.T) {
	var o LeaseOwner
	parent, cancel := context.WithCancel(context.Background())
	l := o.Renew(parent)
	cancel()
	select {
	case <-l.Context().Done():
	default:
		t.Fatal("lease context must follow the parent's cancellation")
	}
	if !l.Valid() {
		t.Fatal("a parent-cancelled lease still owns its generation")
	}
}

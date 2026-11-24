// Package state holds the three primitives the stores share: a published
// immutable view (Published), an ownership lease for async work (LeaseOwner)
// and a classified failure safe for view state (Fault). They replace the
// per-store deep-copy snapshots, ad-hoc generation counters and raw error
// strings described in ARCHITECTURE.md §8.2.
package state

import (
	"context"
	"sync"
	"sync/atomic"
)

// Version is one immutable publication. A published Value must not be
// mutated: writers build the next value by copying only the parts they
// change and sharing the untouched slices and maps.
type Version[T any] struct {
	Rev   uint64
	Value T
}

// Published hands readers the current Version without copying. Publish is
// serialized by an internal lock, so concurrent writers never lose a
// revision and each change channel closes exactly once.
type Published[T any] struct {
	cur     atomic.Pointer[Version[T]]
	mu      sync.Mutex
	changed chan struct{}
}

// Load returns the current version, or nil before the first Publish. It
// never locks and never copies; several frame passes may share the result.
func (p *Published[T]) Load() *Version[T] {
	return p.cur.Load()
}

// Publish stores v as the next revision and closes the channel the last
// Changed call handed out.
func (p *Published[T]) Publish(v T) *Version[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	next := &Version[T]{Rev: 1, Value: v}
	if old := p.cur.Load(); old != nil {
		next.Rev = old.Rev + 1
	}
	p.cur.Store(next)
	if p.changed != nil {
		close(p.changed)
		p.changed = nil
	}
	return next
}

// Changed returns a channel closed when the next Publish happens. Call it
// again after every wake; the channel never carries values.
func (p *Published[T]) Changed() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return p.changed
}

// leaseState is the shared token behind a comparable Lease.
type leaseState struct {
	gen    uint64
	ctx    context.Context
	cancel context.CancelFunc
	valid  atomic.Bool
}

// Lease is a comparable proof of ownership for one generation of async
// work. The zero Lease is invalid.
type Lease struct {
	st *leaseState
}

// Valid reports whether this lease is still the owner's current one.
func (l Lease) Valid() bool {
	return l.st != nil && l.st.valid.Load()
}

// Generation is the owner's sequence number for this lease; 0 is invalid.
func (l Lease) Generation() uint64 {
	if l.st == nil {
		return 0
	}
	return l.st.gen
}

// Context is cancelled when the lease is superseded or retired. Async work
// should both watch it and check Valid before publishing results.
func (l Lease) Context() context.Context {
	if l.st == nil {
		return context.Background()
	}
	return l.st.ctx
}

// LeaseOwner issues leases; every Renew invalidates the previous one, which
// is how a store drops a stale load it already started.
type LeaseOwner struct {
	mu     sync.Mutex
	gen    uint64
	active *leaseState
}

// Renew cancels the previous lease's context and returns the next
// generation's lease, tied to parent.
func (o *LeaseOwner) Renew(parent context.Context) Lease {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.gen++
	ctx, cancel := context.WithCancel(parent)
	st := &leaseState{gen: o.gen, ctx: ctx, cancel: cancel}
	st.valid.Store(true)
	if o.active != nil {
		o.active.valid.Store(false)
		o.active.cancel()
	}
	o.active = st
	return Lease{st: st}
}

// Retire invalidates the current lease; later Renew calls still work.
func (o *LeaseOwner) Retire() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.active != nil {
		o.active.valid.Store(false)
		o.active.cancel()
		o.active = nil
	}
}

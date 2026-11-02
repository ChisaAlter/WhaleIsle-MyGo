package session

import (
	"context"
	"whaleisle/internal/dsh"
)

// Native registrations are compiled equivalents of the fixed original client
// contributions. The Host graph declares their presence; it is never treated as
// a receipt that an unrelated browser mounted React successfully.
type nativeContributionState struct {
	snapshot   dsh.NativeContributions
	owner      *EventsOwner
	cancel     context.CancelFunc
	generation uint64
	ports      [4]uint64
	modelLabel uint64
}

func nativeCommandIndex(name string) int {
	switch name {
	case "file":
		return 0
	case "model":
		return 1
	case "permission":
		return 2
	case "feedback":
		return 3
	default:
		return -1
	}
}

// NativeCommandGeneration changes when this particular registration retires,
// including removal and reinstatement of identical code between UI frames.
func (s ViewState) NativeCommandGeneration(name string) uint64 {
	if index := nativeCommandIndex(name); index >= 0 {
		return s.nativeCommandGenerations[index]
	}
	return 0
}

func (s *Store) setNativeContributionsLocked(snapshot dsh.NativeContributions) {
	s.invalidateViewLocked()
	state := &s.nativeContributions
	if state.snapshot.Known && !snapshot.Known {
		state.generation++
	}
	for index, name := range [...]string{"file", "model", "permission", "feedback"} {
		if state.snapshot.CommandRevision(name) != snapshot.CommandRevision(name) {
			state.ports[index]++
		}
	}
	if state.snapshot.ModelLabelRevision() != snapshot.ModelLabelRevision() || state.snapshot.ModelLabel1017 != snapshot.ModelLabel1017 {
		state.modelLabel++
	}
	state.snapshot = snapshot
}

func (s *Store) resetNativeContributionsLocked() {
	s.invalidateViewLocked()
	if s.nativeContributions.cancel != nil {
		s.nativeContributions.cancel()
	}
	s.nativeContributions = nativeContributionState{generation: s.nativeContributions.generation + 1}
}

// WatchNativeContributionsForEvents starts once for the authenticated, ready
// event generation. Session navigation does not restart this process-level
// registry; individual native actions still capture their session and Host.
func (s *Store) WatchNativeContributionsForEvents(owner *EventsOwner) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.eventsOwnerCurrentLocked(owner) || s.nativeContributions.owner == owner {
		s.mu.Unlock()
		return
	}
	s.resetNativeContributionsLocked()
	ctx, cancel := context.WithCancel(owner.ctx)
	s.nativeContributions.owner, s.nativeContributions.cancel = owner, cancel
	s.mu.Unlock()
	go func() {
		defer cancel()
		err := owner.client.WatchNativeContributions(ctx, func(snapshot dsh.NativeContributions) {
			s.mu.Lock()
			if !s.eventsOwnerCurrentLocked(owner) || ctx.Err() != nil || s.nativeContributions.owner != owner {
				s.mu.Unlock()
				return
			}
			s.setNativeContributionsLocked(snapshot)
			s.mu.Unlock()
			s.fire()
		})
		s.mu.Lock()
		if s.nativeContributions.owner != owner {
			s.mu.Unlock()
			return
		}
		// Also revoke when this context has expired: a canceled stream cannot
		// remain authority merely because no replacement graph has arrived.
		s.setNativeContributionsLocked(dsh.NativeContributions{})
		if err != nil && ctx.Err() == nil {
			s.nativeContributions.snapshot.Error = err.Error()
		}
		s.nativeContributions.cancel = nil
		s.mu.Unlock()
		s.fire()
	}()
}

package session

import (
	"context"
	"whaleisle/internal/dsh"
)

// Native registrations are compiled equivalents of the fixed original client
// contributions. The Host graph declares their presence; it is never treated as
// a receipt that an unrelated browser mounted React successfully.
type nativeContributionState struct {
	snapshot dsh.NativeContributions
	owner    *EventsOwner
	cancel   context.CancelFunc
}

func (s *Store) resetNativeContributionsLocked() {
	if s.nativeContributions.cancel != nil {
		s.nativeContributions.cancel()
	}
	s.nativeContributions = nativeContributionState{}
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
			s.nativeContributions.snapshot = snapshot
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
		s.nativeContributions.snapshot = dsh.NativeContributions{}
		if err != nil && ctx.Err() == nil {
			s.nativeContributions.snapshot.Error = err.Error()
		}
		s.nativeContributions.cancel = nil
		s.mu.Unlock()
		s.fire()
	}()
}

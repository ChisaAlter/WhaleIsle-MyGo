package session

import (
	"whaleisle/internal/dsh"
)

// Native registrations are compiled equivalents of the fixed original client
// contributions. OnEventsReady activates the built-in capability baseline
// (or a one-shot pluginInventory/list snapshot) without any /plugins/events SSE watch.
type nativeContributionState struct {
	snapshot   dsh.NativeContributions
	owner      *EventsOwner
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
	s.nativeContributions = nativeContributionState{generation: s.nativeContributions.generation + 1}
}

func (s *Store) ensureNativeContributionsLocked(owner *EventsOwner) bool {
	if !s.eventsOwnerCurrentLocked(owner) || s.nativeContributions.owner == owner {
		return false
	}
	s.nativeContributions.owner = owner
	s.setNativeContributionsLocked(dsh.DefaultNativeContributions())
	return true
}

// WatchNativeContributionsForEvents ensures native contributions are populated
// once for the authenticated, ready event generation without opening /plugins/events.
//
// Deprecated: OnEventsReady activates native contributions directly; kept temporarily
// until all main entrypoints remove redundant WatchNativeContributionsForEvents calls.
func (s *Store) WatchNativeContributionsForEvents(owner *EventsOwner) {
	if s == nil {
		return
	}
	s.mu.Lock()
	changed := s.ensureNativeContributionsLocked(owner)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

// ResolveNativeContributionsForEvents optionally verifies native contributions via a
// single one-shot pluginInventory/list RPC (no persistent SSE stream) for owner.
func (s *Store) ResolveNativeContributionsForEvents(owner *EventsOwner) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.eventsOwnerCurrentLocked(owner) {
		s.mu.Unlock()
		return
	}
	client, ctx := owner.client, owner.ctx
	s.ensureNativeContributionsLocked(owner)
	s.mu.Unlock()
	go func() {
		snapshot, err := client.ResolveNativeContributions(ctx)
		s.mu.Lock()
		if !s.eventsOwnerCurrentLocked(owner) || ctx.Err() != nil || s.nativeContributions.owner != owner {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.setNativeContributionsLocked(dsh.NativeContributions{Error: faultText("nativeContributions", err)})
		} else {
			s.setNativeContributionsLocked(snapshot)
		}
		s.mu.Unlock()
		s.fire()
	}()
}

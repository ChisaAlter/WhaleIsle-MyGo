package session

import (
	"context"
	"encoding/json"
	"whaleisle/internal/dsh"
)

// AgentInteractionScope is a captured occurrence, never a selected-parent alias.
// Creation only exposes a view; callers must revalidate on every user submit.
type AgentInteractionScope struct {
	pane    *Store
	lease   *agentRetainedRead
	token   uint64
	address AgentSessionAddress
}

func (a *AgentInteractionScope) Client() *dsh.Client      { return a.lease.client }
func (a *AgentInteractionScope) Context() context.Context { return a.lease.ctx }
func (a *AgentInteractionScope) AgentID() string          { return a.address.ChildSessionID }
func (a *AgentInteractionScope) Valid() bool {
	if a == nil || a.pane == nil {
		return false
	}
	s := a.pane
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentInteractionScopeValidLocked(a)
}
func (s *Store) agentInteractionScopeValidLocked(a *AgentInteractionScope) bool {
	n := &s.agentNavigation
	if a.lease == nil || a.lease.restoredReadOnly || a.lease != s.agentRetainedRead || !s.retainedAgentReadValidLocked() || !s.agentNavigationOwnedLocked() ||
		n.view.Token != a.token || n.view.Address != a.address || a.address != a.lease.address || !a.address.validChild() ||
		a.address.Mode != "continuable" || n.view.Busy || !n.view.Process.Live {
		return false
	}
	// Read the original root and direct parent from the current Host registry.
	// Selection of another root neither grants nor revokes this exact occurrence.
	owner := s.agentNavigationOwner
	owner.mu.Lock()
	defer owner.mu.Unlock()
	rootLive, parentLive := false, false
	for _, item := range owner.items {
		if item.SessionID == a.lease.root {
			rootLive = item.AgentAvailable
		}
		if item.SessionID == a.address.ParentSessionID {
			parentLive = item.AgentAvailable
		}
	}
	return rootLive && parentLive
}

// RetainedAgentInteractionScope does not relax AgentInteractionTarget. Restored
// or historical panes without an explicitly acquired live lease cannot use it.
func (s *Store) RetainedAgentInteractionScope(token uint64, address AgentSessionAddress) (*AgentInteractionScope, QuestionProjectionState, []json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := &AgentInteractionScope{pane: s, lease: s.agentRetainedRead, token: token, address: address}
	if !s.agentInteractionScopeValidLocked(a) {
		return nil, QuestionProjectionState{}, nil, false
	}
	_, q, records, _ := s.agentInteractionProjectionLocked(address)
	return a, q, records, true
}

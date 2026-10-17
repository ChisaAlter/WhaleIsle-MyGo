package session

import (
	"context"
	"fmt"
	"whaleisle/internal/dsh"
)

type agentRetainedRead struct {
	restoredReadOnly bool
	restoreVerified  bool
	lineage          []AgentSessionAddress
	ctx              context.Context
	cancel           context.CancelFunc
	pane             *Store
	client           *dsh.Client
	mux              *dsh.Mux
	root             string
	address          AgentSessionAddress
}

// NewRetainedAgentNavigationPane creates a fresh resource occurrence, not a
// persisted pane-ID alias. No reader or capability is opened by creation.
func (s *Store) NewRetainedAgentNavigationPane(label string) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentOccurrenceSequence++
	id := fmt.Sprintf("agent-read:%d:%s", s.agentOccurrenceSequence, label)
	if s.agentNavigationPanes == nil {
		s.agentNavigationPanes = map[string]*Store{}
	}
	pane := New()
	pane.agentNavigationOwner = s
	pane.agentNavigationPaneID = id
	pane.agentRetainEnabled = true
	s.agentNavigationPanes[id] = pane
	return pane
}
func (s *Store) AgentNavigationOccurrenceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentNavigationPaneID
}

// Compare-and-release protects a replacement occurrence from an old tab.
func (s *Store) ReleaseAgentNavigationOccurrence(pane *Store) {
	if pane == nil {
		return
	}
	id := pane.AgentNavigationOccurrenceID()
	s.mu.Lock()
	if s.agentNavigationPanes[id] != pane {
		s.mu.Unlock()
		return
	}
	delete(s.agentNavigationPanes, id)
	if lease := s.agentRetainedReads[id]; lease != nil && lease.pane == pane {
		lease.cancel()
		delete(s.agentRetainedReads, id)
	}
	s.mu.Unlock()
	pane.CloseAgentNavigation()
	pane.mu.Lock()
	pane.client = nil
	pane.mux = nil
	pane.planHistoryBinding = ""
	pane.mu.Unlock()
}
func (s *Store) cancelRetainedAgentReadsLocked(sessionID string) {
	for _, lease := range s.agentRetainedReads {
		if sessionID == "" || lease.root == sessionID || lease.address.ParentSessionID == sessionID || lease.address.ChildSessionID == sessionID || retainedLineageContains(lease.lineage, sessionID) {
			lease.cancel()
		}
	}
}

// Caller holds pane lock; cancellation never locks a pane from its root.
func (s *Store) retainAgentReadLocked(address AgentSessionAddress) {
	if !s.agentRetainEnabled || s.agentNavigationOwner == nil {
		return
	}
	owner := s.agentNavigationOwner
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.agentNavigationPanes[s.agentNavigationPaneID] != s || owner.client != s.client || owner.mux != s.mux || owner.selectedID != s.selectedID || owner.planHistoryBinding != s.planHistoryBinding || owner.followGeneration != s.followGeneration {
		return
	}
	if old := owner.agentRetainedReads[s.agentNavigationPaneID]; old != nil {
		old.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	lease := &agentRetainedRead{ctx: ctx, cancel: cancel, pane: s, client: s.client, mux: s.mux, root: s.selectedID, address: address}
	if owner.agentRetainedReads == nil {
		owner.agentRetainedReads = map[string]*agentRetainedRead{}
	}
	owner.agentRetainedReads[s.agentNavigationPaneID] = lease
	s.agentRetainedRead = lease
}
func (s *Store) retainedAgentReadValidLocked() bool {
	lease, owner := s.agentRetainedRead, s.agentNavigationOwner
	if lease == nil || owner == nil || lease.ctx.Err() != nil || s.agentNavigation.view.Address != lease.address {
		return false
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.agentNavigationPanes[s.agentNavigationPaneID] == s && owner.agentRetainedReads[s.agentNavigationPaneID] == lease && owner.client == lease.client && owner.mux == lease.mux && s.client == lease.client && s.mux == lease.mux && s.selectedID == lease.root
}
func (s *Store) RetainedAgentReadValid() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.retainedAgentReadValidLocked()
}
func (s *Store) agentPaneReadCurrentLocked() bool {
	if s.agentRetainedRead != nil {
		return s.retainedAgentReadValidLocked()
	}
	return s.agentPaneCurrentLocked()
}
func (s *Store) agentReadContextLocked() context.Context {
	if s.agentRetainedRead != nil {
		return s.agentRetainedRead.ctx
	}
	return context.Background()
}

// Only an explicit open under the original selected root may replace a lease
// and acquire fresh action ownership. Merely rendering it never does.
func (s *Store) prepareAgentReadReopen(root string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agentRetainedRead == nil {
		return true
	}
	owner := s.agentNavigationOwner
	if owner == nil {
		return false
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.selectedID != root || owner.agentNavigationPanes[s.agentNavigationPaneID] != s {
		return false
	}
	s.agentRetainedRead.cancel()
	if owner.agentRetainedReads[s.agentNavigationPaneID] == s.agentRetainedRead {
		delete(owner.agentRetainedReads, s.agentNavigationPaneID)
	}
	s.agentRetainedRead = nil
	return true
}

// HasRetainedAgentRead includes revoked leases so render/restore cannot silently
// turn revocation into a fresh authorization. Explicit reopen retires the lease.
func (s *Store) HasRetainedAgentRead() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentRetainedRead != nil
}

func retainedLineageContains(path []AgentSessionAddress, id string) bool {
	for _, edge := range path {
		if edge.ParentSessionID == id || edge.ChildSessionID == id {
			return true
		}
	}
	return false
}

package session

import (
	"context"
	"fmt"
	"time"
)

// RestoreRetainedAgentChat schedules only a fresh read. The durable Host scope
// must match; legacy or nested records lacking a verified root path are refused.
// A provisional, read-only lease owns cancellation during API revalidation.
func (s *Store) RestoreRetainedAgentChat(scope, root string, address AgentSessionAddress, title string) bool {
	return s.RestoreRetainedAgentLineage(scope, root, address, title, []AgentSessionAddress{address})
}
func (s *Store) RestoreRetainedAgentLineage(scope, root string, address AgentSessionAddress, title string, lineage []AgentSessionAddress) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.agentNavigationOwner
	if owner == nil || !s.agentRetainEnabled || s.agentRetainedRead != nil || scope == "" || scope == "legacy" || root == "" || !address.validChild() || !validAgentRestoreLineage(root, address, lineage) {
		return false
	}
	owner.mu.Lock()
	if owner.workspacePreferences.scope != scope || !owner.workspaceReady || owner.client == nil || owner.mux == nil || owner.agentNavigationPanes[s.agentNavigationPaneID] != s {
		owner.mu.Unlock()
		return false
	}
	known := false
	for _, item := range owner.items {
		if item.SessionID == root {
			known = true
			break
		}
	}
	if !known {
		owner.mu.Unlock()
		return false
	}
	s.client, s.mux = owner.client, owner.mux
	s.selectedID, s.planHistoryBinding = root, fmt.Sprintf("%p\x00%s", owner.client, root)
	s.followGeneration = owner.followGeneration
	s.items = append([]Item(nil), owner.items...)
	s.notify = owner.notify
	// No model, permission, draft, delivery selection or old token is restored.
	ctx, cancel := context.WithCancel(context.Background())
	lease := &agentRetainedRead{ctx: ctx, cancel: cancel, pane: s, client: s.client, mux: s.mux, root: root, address: address, restoredReadOnly: true, lineage: append([]AgentSessionAddress(nil), lineage...)}
	if owner.agentRetainedReads == nil {
		owner.agentRetainedReads = map[string]*agentRetainedRead{}
	}
	owner.agentRetainedReads[s.agentNavigationPaneID] = lease
	s.agentRetainedRead = lease
	owner.mu.Unlock()
	s.invalidateAgentNavigationLocked()
	s.agentNavigation = agentNavigationState{client: s.client, mux: s.mux, root: root, binding: s.planHistoryBinding, generation: s.followGeneration, view: AgentNavigationView{Token: s.agentNavigation.view.Token, SessionID: address.ChildSessionID, Address: address, Title: title, Depth: 1, Busy: true}}
	go s.verifyRestoredAgentReader(lease)
	return true
}

func restoredAgentProjectionContains(root string, address AgentSessionAddress, projection agentNavigationProjection) bool {
	v := AgentNavigationView{SessionID: root}
	// Never fall back to a cached global session list for relationship proof.
	if seedAgentNavigationCatalog(&v, projection.Values, projection.AsOfSeq, nil) != nil {
		return false
	}
	for _, row := range v.Rows {
		if row.Address == address && row.CanOpen() {
			return true
		}
	}
	return false
}
func (s *Store) verifyRestoredAgentReader(lease *agentRetainedRead) {
	ctx, cancel := context.WithTimeout(lease.ctx, 15*time.Second)
	defer cancel()
	var err error
	for _, edge := range lease.lineage {
		var projection agentNavigationProjection
		err = lease.client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": edge.ParentSessionID}}, &projection)
		if err != nil {
			break
		}
		if !restoredAgentProjectionContains(edge.ParentSessionID, edge, projection) {
			err = fmt.Errorf("unverified lineage edge")
			break
		}
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
	}
	s.mu.Lock()
	if s.agentRetainedRead != lease || !s.retainedAgentReadValidLocked() || ctx.Err() != nil {
		lease.cancel()
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.agentNavigation.view.Busy = false
		s.agentNavigation.view.Error = "恢复目标未通过来源目录验证 / Restored target failed source catalog verification"
		lease.cancel() // Keep a tombstone; rendering must never silently retry/rebind.
		s.mu.Unlock()
		s.fire()
		return
	}
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
}

package session

import (
	"context"
	"fmt"
	"testing"
	"whaleisle/internal/dsh"
)

func TestRetainedInteractionScopeExactOwnership(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	root.mu.Lock()
	root.selectSessionLocked("other")
	root.planHistoryBinding = fmt.Sprintf("%p\x00other", root.client)
	root.mu.Unlock()
	pane.SyncAgentNavigationPane()
	if _, _, _, ok := pane.AgentInteractionTarget(7, address); ok {
		t.Fatal("legacy gate was widened")
	}
	scope, _, _, ok := pane.RetainedAgentInteractionScope(7, address)
	if !ok || !scope.Valid() || scope.Client() != root.client || scope.AgentID() != "child" || scope.Context() != pane.agentRetainedRead.ctx {
		t.Fatal("exact retained scope unavailable")
	}
	if _, _, _, ok := pane.RetainedAgentInteractionScope(8, address); ok {
		t.Fatal("stale token accepted")
	}
	foreign := address
	foreign.ChildSessionID = "other-child"
	if _, _, _, ok := pane.RetainedAgentInteractionScope(7, foreign); ok {
		t.Fatal("foreign child accepted")
	}
	root.ReleaseAgentNavigationOccurrence(pane)
	if scope.Valid() || scope.Context().Err() != context.Canceled {
		t.Fatal("release did not revoke captured authority")
	}
}
func TestRetainedInteractionScopeRevalidatesEverySubmit(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Store, *Store)
	}{
		{"root unavailable", func(r, p *Store) { r.items[0].AgentAvailable = false }},
		{"root missing", func(r, p *Store) { r.items = nil }},
		{"Host client", func(r, p *Store) { r.client = &dsh.Client{} }},
		{"Host mux", func(r, p *Store) { r.mux = &dsh.Mux{} }},
		{"pane replaced", func(r, p *Store) { r.agentNavigationPanes[p.agentNavigationPaneID] = New() }},
		{"lease replaced", func(r, p *Store) { r.agentRetainedReads[p.agentNavigationPaneID] = &agentRetainedRead{} }},
		{"token changed", func(r, p *Store) { p.agentNavigation.view.Token++ }},
		{"child stopped", func(r, p *Store) { p.agentNavigation.view.Process.Live = false }},
		{"history busy", func(r, p *Store) { p.agentNavigation.view.Busy = true }},
		{"restored read only", func(r, p *Store) { p.agentRetainedRead.restoredReadOnly = true }},
		{"revoked", func(r, p *Store) { p.agentRetainedRead.cancel() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, pane, address := retainedOwnerFixture()
			scope, _, _, ok := pane.RetainedAgentInteractionScope(7, address)
			if !ok {
				t.Fatal("fixture scope")
			}
			tt.change(root, pane)
			if scope.Valid() {
				t.Fatal("old captured callback retained authority")
			}
			if _, _, _, ok = pane.RetainedAgentInteractionScope(7, address); ok {
				t.Fatal("render recreated authority")
			}
		})
	}
}

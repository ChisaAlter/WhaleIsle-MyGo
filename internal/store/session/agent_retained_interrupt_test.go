package session

import "testing"

func TestRetainedAgentExplicitInterruptRejectsWrongOrRevokedTarget(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	root.mu.Lock()
	root.selectSessionLocked("other")
	root.mu.Unlock()
	wrong := address
	wrong.ChildSessionID = "sibling"
	pane.InterruptRetainedAgentExplicit(7, wrong)
	pane.InterruptRetainedAgentExplicit(6, address)
	if pane.agentNavigation.view.ActionBusy {
		t.Fatal("stale token or sibling interrupt started")
	}
	pane.agentRetainedRead.cancel()
	pane.InterruptRetainedAgentExplicit(7, address)
	if pane.agentNavigation.view.ActionBusy {
		t.Fatal("revoked interrupt started")
	}
}

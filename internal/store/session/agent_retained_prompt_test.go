package session

import (
	"context"
	"testing"
)

func TestRetainedAgentExplicitPromptAdmission(t *testing.T) {
	root, pane, address := retainedOwnerFixture()
	root.mu.Lock()
	root.selectSessionLocked("unrelated")
	root.promptMode = PromptModeSteer
	root.mu.Unlock()
	pane.SyncAgentNavigationPane()
	pane.mu.Lock()
	if !pane.retainedAgentPromptAllowedLocked(7, address) {
		t.Fatal("original live owner lost explicit prompt admission")
	}
	if pane.agentNavigationActionAllowedLocked(7, address) {
		t.Fatal("generic action gate was broadened")
	}
	if pane.agentPromptActionAdmissionLocked(6, address, true) {
		t.Fatal("stale token admitted")
	}
	wrong := address
	wrong.ChildSessionID = "another"
	if pane.agentPromptActionAdmissionLocked(7, wrong, true) {
		t.Fatal("different child admitted")
	}
	if pane.promptMode == root.promptMode {
		t.Fatal("borrowed new parent's delivery mode")
	}
	ctx, cancel := context.WithCancel(pane.agentReadContextLocked())
	defer cancel()
	pane.mu.Unlock()
	if _, _, _, ok := pane.AgentInteractionTarget(7, address); ok {
		t.Fatal("shared reply authority broadened")
	}
	root.ReleaseAgentNavigationOccurrence(pane)
	if ctx.Err() == nil {
		t.Fatal("pending action context survived exact release")
	}
	pane.mu.Lock()
	defer pane.mu.Unlock()
	if pane.agentPromptActionAdmissionLocked(7, address, true) {
		t.Fatal("closed lease admitted")
	}
}
func TestRetainedAgentExplicitPromptUnavailableAndRevoked(t *testing.T) {
	for _, scenario := range []string{"unknown-parent", "unavailable-parent", "not-live", "revoked", "host-replaced", "occurrence-replaced"} {
		t.Run(scenario, func(t *testing.T) {
			root, pane, address := retainedOwnerFixture()
			switch scenario {
			case "unknown-parent":
				root.items = nil
			case "unavailable-parent":
				root.items[0].AgentAvailable = false
			case "not-live":
				pane.agentNavigation.view.Process.Live = false
			case "revoked":
				pane.agentRetainedRead.cancel()
			case "host-replaced":
				root.client = nil
			case "occurrence-replaced":
				root.agentNavigationPanes[pane.agentNavigationPaneID] = New()
			}
			pane.mu.Lock()
			defer pane.mu.Unlock()
			if pane.retainedAgentPromptAllowedLocked(7, address) {
				t.Fatal("invalid source admitted explicit send")
			}
		})
	}
}

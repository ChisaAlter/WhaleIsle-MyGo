package session

import "testing"

func TestAgentPaneOwnershipIsolation(t *testing.T) {
	root := New()
	root.selectedID, root.planHistoryBinding = "root", "binding"
	root.agentNavigation.view.Token = 99
	a, b := root.AgentNavigationPane("a"), root.AgentNavigationPane("b")
	a.SyncAgentNavigationPane()
	b.SyncAgentNavigationPane()
	if a == b || a.agentNavigation.view.Token == 99 || len(a.agentNavigation.parents) != 0 {
		t.Fatal("copied main navigation capability")
	}
	for _, pane := range []*Store{a, b} {
		pane.agentNavigation.binding = "binding"
		pane.agentNavigation.root = "root"
	}
	a.agentNavigation.view.Address = AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child-a", Mode: "continuable"}
	b.agentNavigation.view.Address = AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child-b", Mode: "continuable"}
	if !a.agentNavigationOwnedLocked() || b.AgentNavigationSnapshot().Address.ChildSessionID != "child-b" {
		t.Fatal("pane ownership lost")
	}
	root.ReleaseAgentNavigationPane("a")
	if root.AgentNavigationPaneByID("a") != nil || a.agentNavigationOwnedLocked() || !b.agentNavigationOwnedLocked() {
		t.Fatal("release affected wrong owner")
	}
	a.SyncAgentNavigationPane()
	if a.agentPaneCurrentLocked() {
		t.Fatal("released owner revived")
	}
	root.selectedID = "other"
	if b.agentNavigationOwnedLocked() {
		t.Fatal("old root allowed action before render sync")
	}
}

package session

import "testing"

func TestAgentChatTargetIsAddressedAndRevocable(t *testing.T) {
	root := New()
	root.selectedID, root.planHistoryBinding = "root", "binding"
	pane := root.AgentNavigationPane("catalog")
	pane.SyncAgentNavigationPane()
	pane.agentNavigation.binding, pane.agentNavigation.root = "binding", "root"
	address := AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "continuable"}
	pane.agentNavigation.view = AgentNavigationView{Token: 7, SessionID: "root", Rows: []AgentNavigationRow{{Address: address, Label: "Child"}}}
	if owner, title, ok := pane.AgentChatTarget(7, address); !ok || owner != "root" || title != "Child" {
		t.Fatal("catalog target lost", owner, title, ok)
	}
	if _, _, ok := pane.AgentChatTarget(6, address); ok {
		t.Fatal("stale token authorized chat")
	}
	altered := address
	altered.Mode = "one-shot"
	if _, _, ok := pane.AgentChatTarget(7, altered); ok {
		t.Fatal("altered routing authorized")
	}
	root.ReleaseAgentNavigationPane("catalog")
	if _, _, ok := pane.AgentChatTarget(7, address); ok {
		t.Fatal("closed catalog authorized chat")
	}
	if pane.OpenAgentChat("root", address, "Child") {
		t.Fatal("released reader reopened")
	}
}

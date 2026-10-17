package frame

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestSubagentChatHiddenRetainsUntilClose(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("subagentchat")
	owner := session.New()
	n := w.Root
	n.SubagentChat = &SubagentChatPaneState{Root: "root"}
	nav := owner.AgentNavigationPane(n.ID)
	n.Agents = &WorkspaceAgentState{Owner: owner, Navigation: nav, Binding: "binding"}
	w.BeginContentFrame("binding", "root")
	w.EndContentFrame()
	if owner.AgentNavigationPaneByID(n.ID) != nav {
		t.Fatal("hidden child chat released its live occurrence")
	}
	w.close(n.ID)
	if owner.AgentNavigationPaneByID(n.ID) != nil {
		t.Fatal("closed child chat retained reader")
	}
}

func TestSubagentChatHiddenBindingChangeReleases(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("subagentchat")
	owner := session.New()
	n := w.Root
	n.SubagentChat = &SubagentChatPaneState{Root: "root"}
	n.Agents = &WorkspaceAgentState{Owner: owner, Navigation: owner.AgentNavigationPane(n.ID), Binding: "old"}
	w.BeginContentFrame("new", "other")
	w.EndContentFrame()
	if owner.AgentNavigationPaneByID(n.ID) != nil {
		t.Fatal("changed ownership retained hidden reader")
	}
}

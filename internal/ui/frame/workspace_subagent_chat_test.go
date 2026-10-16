package frame

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestSubagentChatSplitRestoreAndClose(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("subagentchat")
	address := session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "child", Mode: "one-shot"}
	w.Root.SubagentChat = &SubagentChatPaneState{Root: "root", Address: address, Title: "Child"}
	original := w.Root.ID
	owner := session.New()
	w.Root.Agents = &WorkspaceAgentState{Owner: owner, Navigation: owner.AgentNavigationPane(original)}
	tab := "subagentchat"
	m := Model{RightPaneOpen: true, RightPaneTab: &tab, Sizes: &LayoutSizes{Workspace: w}}
	SplitWorkspacePane(m, false)
	if w.Root.Second.SubagentChat == nil || w.Root.Second.SubagentChat == w.Root.First.SubagentChat || w.Root.Second.SubagentChat.Address != address {
		t.Fatal("split lost or shared routing state")
	}
	if w.Root.Second.Agents != nil {
		t.Fatal("split inherited live capability")
	}
	restored, err := RestoreWorkspacePanes(w.Export())
	if err != nil {
		t.Fatal(err)
	}
	n := restored.Root.find(original)
	if n.SubagentChat.Address != address || n.Agents != nil {
		t.Fatal("restoration lost address or retained reader")
	}
	w.close(original)
	if owner.AgentNavigationPaneByID(original) != nil {
		t.Fatal("closed chat retained reader")
	}
	restored.close(original)
	if restored.Root.Kind != "subagentchat" || restored.Root.SubagentChat.Address != address {
		t.Fatal("closing chat disturbed sibling")
	}
}

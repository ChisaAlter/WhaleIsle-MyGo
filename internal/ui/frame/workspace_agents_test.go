package frame

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestAgentPaneIndependentState(t *testing.T) {
	w := &WorkspacePanes{}
	kind := "agents"
	w.syncRoute(kind)
	root := session.New()
	m := Model{Store: root, RightPaneOpen: true, RightPaneTab: &kind, Sizes: &LayoutSizes{Workspace: w}}
	a := w.Root
	first := w.agentPaneModel(m, a)
	*first.AgentsPromptDraft = "private draft"
	SplitWorkspacePane(m, false)
	b := w.Root.Second
	if b.Kind != "agents" || b.Agents != nil {
		t.Fatal("split copied capability or draft")
	}
	second := w.agentPaneModel(m, b)
	if first.AgentNavigationStore == second.AgentNavigationStore || *second.AgentsPromptDraft != "" {
		t.Fatal("shared navigation or draft")
	}
	a.releaseAgentPane()
	if root.AgentNavigationPaneByID(a.ID) != nil || root.AgentNavigationPaneByID(b.ID) != second.AgentNavigationStore {
		t.Fatal("closed wrong source")
	}
}

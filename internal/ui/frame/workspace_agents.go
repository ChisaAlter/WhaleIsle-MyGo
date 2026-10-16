package frame

import "whaleisle/internal/store/session"

// Nothing in this state is restored as an active child capability. A new/split
// pane starts at the root catalog; only explicit user clicks enter a child.
type WorkspaceAgentState struct {
	Owner      *session.Store
	Navigation *session.Store
	Manager    *session.AgentPresetManager
	Binding    string
	Prompt     string
	rendered   bool
}

func (w *WorkspacePanes) agentPaneModel(m Model, n *WorkspacePane) Model {
	if m.Store == nil {
		return m
	}
	if n.Agents == nil {
		n.Agents = &WorkspaceAgentState{}
	}
	a := n.Agents
	binding := m.Store.Snapshot().PlanHistoryBinding
	if a.Owner != m.Store || a.Binding != binding {
		n.releaseAgentPane()
		*a = WorkspaceAgentState{Owner: m.Store, Binding: binding, Manager: session.NewAgentPresetManager(m.Store)}
	}
	a.Navigation = m.Store.AgentNavigationPane(n.ID)
	a.Navigation.SyncAgentNavigationPane()
	a.rendered = true
	m.AgentNavigationStore = a.Navigation
	m.AgentsPromptDraft = &a.Prompt
	m.AgentPresetManager = a.Manager
	open := m.OnOpenAgentPaneDocument
	navigation := a.Navigation
	m.OnOpenAgentDocument = nil
	if open != nil {
		m.OnOpenAgentDocument = func(token uint64, address session.AgentSessionAddress, path string) {
			w.Active = n.ID
			open(n.ID, navigation, token, address, path)
		}
	}
	return m
}
func (n *WorkspacePane) releaseAgentPane() {
	if n.Agents != nil && n.Agents.Owner != nil {
		n.Agents.Owner.ReleaseAgentNavigationPane(n.ID)
		n.Agents.Navigation = nil
	}
}

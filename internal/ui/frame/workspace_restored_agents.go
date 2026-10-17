package frame

import "whaleisle/internal/store/session"

// Runs for saved layouts, including never-painted tabs. Only the original
// Host-scoped layout and direct-root address can initiate revalidation.
func (w *WorkspacePanes) RestoreRetainedAgentReaders(owner *session.Store, scope, root string) {
	if w == nil || owner == nil || scope == "" || scope == "legacy" {
		return
	}
	w.Root.eachResource(func(n *WorkspacePane) {
		target := n.SubagentChat
		if n.Kind != "subagentchat" || target == nil || target.Root != root || target.Address.ParentSessionID != root {
			return
		}
		if n.Agents != nil && n.Agents.Navigation != nil {
			return
		}
		pane := owner.NewRetainedAgentNavigationPane(n.ID)
		if !pane.RestoreRetainedAgentChat(scope, root, target.Address, target.Title) {
			owner.ReleaseAgentNavigationOccurrence(pane)
			return
		}
		n.Agents = &WorkspaceAgentState{Owner: owner, Navigation: pane, Binding: pane.Snapshot().PlanHistoryBinding, Manager: session.NewAgentPresetManager(owner)}
	})
}

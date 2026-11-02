package frame

import "whaleisle/internal/store/session"

// Runs for saved layouts, including never-painted tabs. Only the original
// Host-scoped layout and an explicit root-to-child ID chain can initiate revalidation.
func (w *WorkspacePanes) RestoreRetainedAgentReaders(owner *session.Store, scope, root string) {
	if w == nil || owner == nil || scope == "" || scope == "legacy" {
		return
	}
	w.Root.eachResource(func(n *WorkspacePane) {
		target := n.SubagentChat
		if n.Kind != "subagentchat" || target == nil || target.Root != root {
			return
		}
		if n.Agents != nil && n.Agents.Navigation != nil {
			return
		}
		pane := owner.NewRetainedAgentNavigationPane(n.ID)
		lineage := target.Lineage
		if len(lineage) == 0 && target.Address.ParentSessionID == root {
			lineage = []session.AgentSessionAddress{target.Address}
		}
		if !pane.RestoreRetainedAgentLineage(scope, root, target.Address, target.Title, lineage) {
			owner.ReleaseAgentNavigationOccurrence(pane)
			return
		}
		n.Agents = &WorkspaceAgentState{Owner: owner, Navigation: pane, Binding: pane.PlanHistoryBinding(), Manager: session.NewAgentPresetManager(owner)}
	})
}

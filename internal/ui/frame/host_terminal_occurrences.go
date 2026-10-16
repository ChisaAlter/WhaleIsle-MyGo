package frame

func observeHostTerminalPanes(m Model) {
	var keys []string
	w := m.Sizes.Workspace
	w.Root.eachLeaf(func(group *WorkspacePane) {
		if m.RightPaneOpen {
			if docked := group.dockedLeaf(); docked != nil && docked.Kind == "host-terminal" {
				keys = append(keys, docked.ID)
			}
		}
		for _, n := range group.dockTabs() {
			if n.Kind == "host-terminal" && n.Float != nil && !n.Float.Minimized {
				keys = append(keys, n.ID)
			}
		}
	})
	m.HostTerminalUI.ReconcileOccurrences(m.HostTerminalStore, keys)
}
func hostTerminalPaneModel(m Model) Model {
	if m.workspacePane == nil {
		return m
	}
	pane := m.workspacePane
	session := m.Store.Snapshot().SelectedID
	m.HostTerminalStore = m.HostTerminalStore.Occurrence(pane.ID, pane.TerminalSource, session)
	m.HostTerminalUI = m.HostTerminalUI.Occurrence(pane.ID)
	return m
}

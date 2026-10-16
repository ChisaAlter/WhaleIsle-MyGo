package frame

func observeHostTerminalPanes(m Model) {
	var keys []string
	var visit func(*WorkspacePane)
	visit = func(n *WorkspacePane) {
		if n == nil {
			return
		}
		if n.First != nil {
			visit(n.First)
			visit(n.Second)
			return
		}
		if n.Kind == "host-terminal" {
			keys = append(keys, n.ID)
		}
	}
	if m.RightPaneOpen {
		w := m.Sizes.Workspace
		n := w.Root
		if w.Fullscreen != "" {
			n = w.Root.find(w.Fullscreen)
		}
		visit(n)
	}
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

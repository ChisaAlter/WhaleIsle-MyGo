package frame

import "whaleisle/internal/store/hostterminal"

func observeHostTerminalPanes(m Model) {
	var keys []string
	w := m.Sizes.Workspace
	// Resolve completed close requests on the UI thread, before enumerating panes.
	var completed []string
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.terminalCloseRequest != nil {
			if done, _ := n.terminalCloseRequest.Result(); done {
				completed = append(completed, n.ID)
			}
		}
	})
	for _, id := range completed {
		w.close(id)
	}

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
	snapshot := m.Store.SelectionSummary()
	session := snapshot.SelectedID
	if pane.terminalFresh {
		generation := snapshot.ComposerHostGeneration
		if !pane.terminalFreshBound {
			pane.terminalFreshBound, pane.terminalFreshGeneration, pane.terminalFreshSession = true, generation, session
		} else if pane.terminalFreshGeneration != generation || pane.terminalFreshSession != session {
			pane.terminalFresh = false
		}
	}
	root, renderer := m.HostTerminalStore, m.HostTerminalUI
	m.HostTerminalStore = root.Occurrence(pane.ID, pane.TerminalSource, session)
	child := m.HostTerminalStore
	owner := child.Snapshot(session).Owner
	authority := m.Store.WorkspaceLayoutScope()
	child.AdoptRecovery(owner, authority, session, pane.TerminalRecovery)
	pane.terminalRecovery = func() *hostterminal.RecoveryTarget { return child.RecoverySnapshot(authority, session) }
	if saved := pane.terminalRecovery(); saved != nil {
		pane.TerminalRecovery = saved
	}
	pane.terminalClose = func() *hostterminal.OccurrenceClose { return root.BeginOccurrenceClose(pane.ID, child, owner, session) }
	pane.terminalDispose = func() { renderer.ForgetOccurrence(pane.ID) }
	m.HostTerminalUI = m.HostTerminalUI.Occurrence(pane.ID)
	return m
}

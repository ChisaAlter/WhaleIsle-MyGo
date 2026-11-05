package frame

import "github.com/egoist/mygo/ui"

// Focus, not the last selected dock tab, determines a page command target.
// Committed bindings also resolve descendants without construction-order races.
func focusedWorkspacePage(c *ui.Context, m Model) *WorkspacePane {
	if m.WorkspacePage || m.Sizes == nil || m.Sizes.Workspace == nil {
		return nil
	}
	w := m.Sizes.Workspace
	n := w.Root.find(ui.FocusedValue(c, &w.focusedResource))
	if n == nil || n.First != nil {
		return nil
	}
	if n.Float != nil {
		if n.Float.Minimized {
			return nil
		}
		return n
	}
	if !m.RightPaneOpen {
		return nil
	}
	group := w.Root.dockGroup(n.ID)
	if group == nil || group.dockedLeaf() != n {
		return nil
	}
	return n
}

func splitFocusedWorkspacePage(c *ui.Context, m Model) {
	if n := focusedWorkspacePage(c, m); n != nil && n.Float == nil {
		w := m.Sizes.Workspace
		w.activateDockTab(n.ID)
		if m.RightPaneTab != nil {
			*m.RightPaneTab = n.Kind
		}
		splitWorkspaceDock(m, false)
	}
}

func maximizeFocusedWorkspacePage(c *ui.Context, m Model) {
	if n := focusedWorkspacePage(c, m); n != nil {
		workspacePaneMaximize(m, n)
	}
}

func closeFocusedWorkspacePage(c *ui.Context, m Model) {
	if m.OnCloseForegroundModal != nil && m.OnCloseForegroundModal() {
		return
	}
	// Unknown dialogs/menus retain ownership rather than closing a background
	// resource or the window. The top modal's own owner handles its close.
	if !c.MenuOverlayClear() {
		return
	}
	if n := focusedWorkspacePage(c, m); n != nil && m.Sizes.Workspace.canCloseResource(n) {
		workspacePaneClose(m, n)
		return
	}
	if m.OnCloseWindow != nil {
		m.OnCloseWindow()
	}
}

func refreshFocusedWorkspacePage(c *ui.Context, m Model) {
	if !c.MenuOverlayClear() {
		return
	}
	if n := focusedWorkspacePage(c, m); n != nil && n.refreshResource != nil {
		n.refreshResource()
	}
}

// Install only an occurrence's real refresh capability. The captured reader
// cannot run after its Host/session/workspace binding has changed.
func bindWorkspacePageRefresh(m Model, refresh func()) {
	n := m.workspacePane
	if n == nil || m.Store == nil || m.Sizes == nil || m.Sizes.Workspace == nil {
		return
	}
	w, owner := m.Sizes.Workspace, m.Store
	binding := owner.PlanHistoryBinding()
	n.refreshResource = func() {
		if w.Root.find(n.ID) != n || owner.PlanHistoryBinding() != binding {
			return
		}
		refresh()
	}
}

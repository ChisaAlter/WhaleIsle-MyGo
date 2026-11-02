package frame

import "whaleisle/internal/ui/planpreview"

// The existing checked open callback owns authorization and history lookup.
// Switch chrome only after it has opened this exact resource in this pane.
func (w *WorkspacePanes) showOpenedPlan(m Model, doc planpreview.Document) {
	if m.Store == nil {
		return
	}
	n := w.Root.find(w.Active)
	if n == nil || n.Files == nil {
		return
	}
	sid := m.Store.SelectedSessionID()
	if n.Files.Tabs.Plans.ActiveAddress(sid) != doc.Address {
		return
	}
	target := w.selectKind(n, "plan")
	transferDockResource(target, n, "plan")
	if m.RightPaneTab != nil {
		*m.RightPaneTab = "plan"
	}
}

func copyPlanPane(target, source *WorkspacePane) {
	if target.Kind != "plan" || source.Kind != "plan" || source.Files == nil {
		return
	}
	target.Files = &WorkspaceFileState{Tabs: source.Files.Tabs.CopyPlanPane()}
}

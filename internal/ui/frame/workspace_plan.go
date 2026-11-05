package frame

import (
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/planpreview"
)

// OpenPlanResource keeps the authorized plan address in its own occurrence,
// shared by chat plan cards, child conversations and explicit file navigation.
func (w *WorkspacePanes) OpenPlanResource(sessionID, binding, root string, doc planpreview.Document) string {
	var target *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.Kind == "plan" && n.Files != nil && n.Files.previewBinding == binding && n.Files.previewRoot == root && n.Files.Tabs.Plans.ActiveAddress(sessionID) == doc.Address {
			target = n
		}
	})
	added := target == nil
	if target == nil {
		target = w.leaf("plan")
		target.Files = &WorkspaceFileState{Mode: files.EditViewEdit, Expanded: map[string]bool{}, previewBinding: binding, previewRoot: root}
		target.Files.Tabs.Bind(binding, root)
	}
	if err := target.Files.Tabs.OpenPlanResource(sessionID, binding, doc); err != "" {
		return err
	}
	target.Files.ResourceTitle = doc.Title
	sourceID := w.Active
	w.pending = func() {
		source := w.Root.find(sourceID)
		if added {
			w.placeDockResource(target, source, false)
		}
		w.activateDockTab(target.ID)
		w.focusPending = target.ID
		if target.Float != nil {
			w.MinimizeWorkspaceFloat(target.ID, false)
		}
		if source != nil && source.Kind == "guide" {
			w.disposeWorkspaceResource(source.ID)
		}
	}
	return ""
}

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
	target.Files = &WorkspaceFileState{Tabs: source.Files.Tabs.CopyPlanPane(), ResourceTitle: source.Files.ResourceTitle}
}

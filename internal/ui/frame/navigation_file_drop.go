package frame

import "github.com/egoist/mygo/ui"

func navigationFileDropAllowed(m Model) bool {
	if m.Store == nil || m.Draft == nil || m.OnAttachComposerFiles == nil {
		return false
	}
	return m.Sizes == nil || m.Sizes.Workspace == nil || m.Sizes.Workspace.currentMainChild(m) == nil
}

// Register only the navigation rail. Resource docks, WebViews and child panes
// remain outside this ancestry and keep their own native drop destinations.
func navigationFileDropRail(c *ui.Context, m Model, content func()) {
	area := ui.Column(c.Key("navigation-files")).Fill().MinHeight(0)
	area.Children(func() {
		if navigationFileDropAllowed(m) {
			conversationFileDrop(c, m, area, m.Store.Snapshot().SelectedID)
		}
		content()
	})
}

package frame

import "github.com/egoist/mygo/ui"

// NavigationShell owns the persistent workspace rail. Chat and global panels
// occupy the same center slot; fullscreen settings/guest pages bypass this API.
// Stores, draft pointers and split preferences remain owned by the caller.
func NavigationShell(c *ui.Context, m Model, content func()) {
	if m.Store == nil || m.Sizes == nil {
		content()
		return
	}
	viewport, _ := c.Size()
	m.Sizes.ResolveColumns(viewport, m.RightPaneOpen)
	registerPageCommands(c, m)
	ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinHeight(0).Children(func() {
		sidebarColumns(c, m.Sizes, func() { navigationView(c, m, m.Store.Snapshot()) }, content)
	})
}

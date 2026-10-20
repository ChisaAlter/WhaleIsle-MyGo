package frame

import (
	"github.com/egoist/mygo/ui"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// NavigationShell owns the persistent workspace rail. Chat and global panels
// occupy the same center slot; fullscreen settings/guest pages bypass this API.
// Stores, draft pointers and split preferences remain owned by the caller.
func NavigationShell(c *ui.Context, m Model, content func()) {
	if m.Store == nil || m.Sizes == nil {
		content()
		return
	}
	viewport, _ := viewportDIP(c)
	m.Sizes.ResolveColumns(viewport, m.RightPaneOpen)
	registerPageCommands(c, m)
	ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinHeight(0).Children(func() {
		sidebarColumns(c, m.Sizes, func() { navigationFileDropRail(c, m, func() { navigationView(c, m, m.Store.Snapshot()) }) }, func() {
			if !m.WorkspacePage {
				content()
				return
			}
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			ui.Column(c.Key("global-workspace-rows")).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				workspaceHeaderBand(c, func() { workspaceSharedCaption(c, m, m.Store.Snapshot(), colors, true) })
				centerTerminalColumn(c, m, content, func() {
					termui.View(c, termui.Model{
						Panel: m.Terminal, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
						ShellsNote: m.TerminalShellsNote, OnRestart: m.OnRestartTerminal, OnClose: m.OnCloseTerminal,
						OnAddToChat: m.OnAddSelectionToChat, CanAddToChat: m.CanAddSelectionToChat,
						TargetBinding: m.FileBindingID, OnOpenTarget: m.OnOpenTerminalTarget,
					})
				})
			})
		})
	})
}

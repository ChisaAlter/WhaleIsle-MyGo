package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
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
	viewport, height := viewportDIP(c)
	m.ViewportW, m.ViewportH = viewport, height
	m.Sizes.observeSidebarGeometry(viewport, height)
	m.Sizes.ObserveSidebarSessionSelection(renderSnapshot(m))
	m.Sizes.resolveColumnsWithReserve(viewport, m.RightOverlayReserve, m.RightPaneOpen)
	registerPageCommands(c, m)
	// The production frame lives inside this shell with ExternalNavigation.
	// A phone rightbar covers the viewport rather than the rail's clipped main
	// column. Global workspace pages keep their own navigation and caption.
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	canvas := chrome.BackdropCanvas(chrome.Color(colors.BG))
	if !m.WorkspacePage && workspaceAutoFullscreen(m) {
		ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinWidth(0).MinHeight(0).Background(canvas).Children(func() {
			content()
			renderGlobalShellSlot(c, m, "shell.bottom", nil)
		})
		return
	}
	ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinHeight(0).Background(canvas).Children(func() {
		sidebarColumns(c, m.Sizes, func() { navigationFileDropRail(c, m, func() { navigationView(c, m, renderSnapshot(m)) }) }, func() {
			if !m.WorkspacePage {
				content()
				return
			}
			colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
			ui.Column(c.Key("global-workspace-rows")).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				workspaceHeaderBand(c, func() { workspaceSharedCaption(c, m, renderSnapshot(m), colors, true) })
				ui.Box(c.Key("global-workspace-body")).Grow(1).MinWidth(0).MinHeight(0).
					Padding(0, maxf(0, m.RightOverlayReserve), 0, 0).Children(func() {
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
		renderGlobalShellSlot(c, m, "shell.bottom", nil)
	})
}

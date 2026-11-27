package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

func frameSplitBackdropColumns(m Model) bool {
	return chrome.BackdropActive() && theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark).Skin != theme.SkinCodex
}

func frameCanvasFill(t *design.Tokens) ui.Color {
	if t.Glass {
		return t.Background
	}
	if chrome.BackdropActive() {
		return chrome.BackdropCanvas(t.Background)
	}
	return t.Background
}

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
	m.Sizes.observeSidebarGeometry(c, viewport, height)
	m.Sizes.ObserveSidebarSessionSelection(renderSnapshot(m))
	m.Sizes.resolveColumnsWithReserve(viewport, m.RightOverlayReserve, m.RightPaneOpen)
	registerPageCommandsPtr(c, &m)
	// The production frame lives inside this shell with ExternalNavigation.
	// A phone rightbar covers the viewport rather than the rail's clipped main
	// column. Global workspace pages keep their own navigation and caption.
	canvas := frameCanvasFill(tk(c))
	splitBackdrop := frameSplitBackdropColumns(m)
	shellBg := canvas
	if splitBackdrop {
		// Under a live wallpaper/gradient, never paint the canvas fill on the
		// outer shell behind the sidebar and right pane; each column paints its
		// own single translucent layer so surfaces never double-stack white fog.
		shellBg = ui.Transparent
	}
	if !m.WorkspacePage && workspaceAutoFullscreen(m) {
		ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinWidth(0).MinHeight(0).Background(shellBg).Children(func() {
			content()
		})
		return
	}
	mp := &m
	ui.Column(c).Key("workspace-navigation-shell").Grow(1).MinHeight(0).Background(shellBg).Children(func() {
		sidebarColumns(c, mp.Sizes, func() { navigationFileDropRail(c, *mp, func() { navigationViewPtr(c, mp, renderSnapshot(*mp)) }) }, func() {
			if !mp.WorkspacePage {
				content()
				return
			}
			colors := theme.Resolve(mp.ThemeID).Active(mp.SystemDark)
			ui.Column(c.Key("global-workspace-rows")).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				workspaceHeaderBand(c, func() { workspaceSharedCaptionPtr(c, mp, renderSnapshot(*mp), colors, true) })
				body := ui.Box(c.Key("global-workspace-body")).Grow(1).MinWidth(0).MinHeight(0).
					Padding(0, maxf(0, mp.RightOverlayReserve), 0, 0)
				if splitBackdrop {
					body.Background(canvas)
				}
				body.Children(func() {
					centerTerminalColumnPtr(c, mp, content, func() {
						termui.View(c, termui.Model{
							Panel: mp.Terminal, ThemeID: mp.ThemeID, SystemDark: mp.SystemDark,
							ShellsNote: mp.TerminalShellsNote, OnRestart: mp.OnRestartTerminal, OnClose: mp.OnCloseTerminal,
							OnAddToChat: mp.OnAddSelectionToChat, CanAddToChat: mp.CanAddSelectionToChat,
							TargetBinding: mp.FileBindingID, OnOpenTarget: mp.OnOpenTerminalTarget,
						})
					})
				})
			})
		})
	})
}

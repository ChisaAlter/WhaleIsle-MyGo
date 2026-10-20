package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func sessionActionsLabel(id string) string {
	return workspaceText("会话操作 ", "Session actions ") + id
}

// rowActionGlyph is one hover action: a bare 16px glyph button
// (Rows.module.css .iconButton) that reaches the same callbacks as the row menu.
// Nothing renders without an owner for the action.
func sessionRowAction(c *ui.Context, key string, art *ui.SVG, label, tip string, ink ui.Color, run func()) {
	if run == nil || art == nil {
		return
	}
	button := ui.Button(c.Key("session-action-"+key), "").Label(label).Tooltip(tip).
		Width(16).Height(16).Padding(0).BorderWidth(0).Background(ui.Transparent).Center()
	button.Children(func() { ui.Icon(c, art).Size(14, 14).TextColor(ink) })
	if button.Clicked() {
		run()
	}
}

func sessionPinLabel(pinned bool) string {
	if pinned {
		return i18n.T(i18n.KeySidebarUnpin)
	}
	return i18n.T(i18n.KeySidebarPin)
}

func sessionPinTip(pinned bool) string {
	if pinned {
		return workspaceText("取消置顶", "Unpin session")
	}
	return workspaceText("置顶会话", "Pin session")
}

func pinAction(m Model, it session.Item) func() {
	if it.Pinned {
		if m.OnUnpin == nil {
			return nil
		}
		return func() { m.OnUnpin(it.SessionID) }
	}
	if m.OnPin == nil {
		return nil
	}
	return func() { m.OnPin(it.SessionID) }
}

func archiveAction(m Model, it session.Item) func() {
	if m.OnArchive == nil {
		return nil
	}
	return func() { m.OnArchive(it.SessionID) }
}

// sessionHoverCard is the original row's hover card: full title, relative time,
// workspace path and every resting status, in the shared menu material.
func sessionHoverCard(c *ui.Context, m Model, anchor ui.Element, it session.Item, label, status string, colors theme.Colors) {
	ui.TooltipBase(c.Key("session-hover:"+it.SessionID), anchor, func(tip ui.Element) {
		width, height := m.viewport(c)
		tip.Label("Session preview: " + it.SessionID).
			Width(min(float32(280), max(float32(0), width-24))).MaxHeight(max(float32(0), height-24)).Clip().
			Padding(12).Radius(12).Background(chrome.Color(colors.BG)).Border(.5, chrome.Color(colors.Border)).
			Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, .16)).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
				ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(chrome.Color(colors.FG)).
					FillWidth().MinWidth(0).MaxLines(3)
				ui.Text(c, sessionTrailingLabel(status, it.UpdatedAt)).FontSize(theme.InterfaceSize(11)).
					TextColor(chrome.Color(colors.Muted)).FillWidth().MinWidth(0)
				if it.Cwd != "" {
					ui.Text(c, it.Cwd).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted)).
						FillWidth().MinWidth(0).MaxLines(2)
				}
				if text := statusLabel(status); text != "" {
					ui.Text(c, text).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted)).
						FillWidth().MinWidth(0)
				}
			})
		})
	})
}

// The native menu handles pointer dismissal, arrows, Escape and focus return.
// Bind actions to the row's ID, never to whichever session is currently open.
func sessionRowMenu(m Model, it session.Item) func(*ui.Menu) {
	return func(menu *ui.Menu) {
		if it.Pinned {
			if m.OnUnpin != nil && menu.Item(i18n.T(i18n.KeySidebarUnpin)).Chosen() {
				m.OnUnpin(it.SessionID)
			}
		} else if m.OnPin != nil && menu.Item(i18n.T(i18n.KeySidebarPin)).Chosen() {
			m.OnPin(it.SessionID)
		}
		if m.OnBeginRename != nil && menu.Item(i18n.T(i18n.KeySidebarRename)).Chosen() {
			m.OnBeginRename(it.SessionID, it.Title)
		}
		if m.OnMoveSessionBefore != nil && menu.Item(i18n.T(i18n.KeySidebarMoveUp)).Chosen() {
			m.OnMoveSessionBefore(it.SessionID, "")
		}
		if m.OnFork != nil && menu.Item(i18n.T(i18n.KeySidebarFork)).Chosen() {
			m.OnFork(it.SessionID)
		}
		if m.OnCopySessionID != nil && menu.Item(i18n.T(i18n.KeySidebarCopyID)).Chosen() {
			m.OnCopySessionID(it.SessionID)
		}
		if m.OnArchive != nil && menu.Item(i18n.T(i18n.KeySidebarArchive)).Chosen() {
			m.OnArchive(it.SessionID)
		}
	}
}

func sessionStatusSlot(c *ui.Context, status string, muted ui.Color) {
	slot := ui.Box(c).Width(16).Height(20).Label(statusLabel(status))
	if status == "" {
		return
	}
	ink := muted
	if status == "running" {
		slot.AlignItems(ui.Center).Justify(ui.Center).Children(func() { ui.Spinner(c).Width(14).Height(14).TextColor(muted) })
		return
	}
	if status == "completed" {
		ink = c.Theme().Success
	} else {
		ink = c.Theme().Warning
	}
	slot.Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.Circle(r.X+8, r.Y+10, 3)
		p.FillPath(path, ink)
	})
}

// Archived rows are inert: restoring and deleting remain explicit menu actions.
func rowArchivedSession(c *ui.Context, id, title string, m Model, muted ui.Color) {
	row := ui.Row(c.Key("archived-session-"+id)).FillWidth().Height(32).Padding(0, 8, 0, 8).Gap(0).AlignItems(ui.Center).Radius(12)
	if row.Hovered() || row.FocusWithin() {
		row.Background(selectedBackground(theme.Resolve(m.ThemeID).Active(m.SystemDark)))
	}
	menu := func(menu *ui.Menu) {
		if m.OnUnarchive != nil && menu.Item(i18n.T(i18n.KeySidebarUnarchive)).Chosen() {
			m.OnUnarchive(id)
		}
		if m.OnDelete != nil && menu.Item(i18n.T(i18n.KeySidebarDelete)).Chosen() {
			m.OnDelete(id)
		}
	}
	row.ContextMenu(menu).Children(func() {
		ui.Box(c).Width(16).Height(20)
		ui.Text(c, title).MinWidth(0).Grow(1).Padding(0, 6, 0, 4).SingleLine().FontSize(theme.InterfaceSize(14)).TextColor(muted)
		more := ui.Button(c, "…").Label(sessionActionsLabel(id)).Width(16).Height(20).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Menu(menu)
		if !row.Hovered() && !row.FocusWithin() && !more.Focused() {
			more.Opacity(0)
		}
	})
}

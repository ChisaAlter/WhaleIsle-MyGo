package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func sessionActionsLabel(id string) string {
	return workspaceText("会话操作 ", "Session actions ") + id
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
	row := ui.Row(c.Key("archived-session-"+id)).FillWidth().Height(32).Padding(0, 8, 0, 8).Gap(0).AlignItems(ui.Center).Radius(8)
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

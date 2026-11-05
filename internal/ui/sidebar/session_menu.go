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

// The native menu handles pointer dismissal, arrows, Escape and focus return.
// Bind actions to the row's ID, never to whichever session is currently open.
func sessionRowMenu(m Model, it session.Item) func(*ui.Menu) {
	return func(menu *ui.Menu) {
		menu.CloseOnPointerLeave(true)
		if it.SessionID == displayedSessionID(m) {
			if m.OnOpenSessionPlugins != nil && menu.Item(i18n.T(i18n.KeyCmdSessionPluginTitle)).Chosen() {
				m.OnOpenSessionPlugins(it.SessionID)
			}
			if m.SessionPluginsActive && m.OnCloseSessionPlugins != nil && menu.Item(i18n.T(i18n.KeySessionPluginsStop)).Chosen() {
				m.OnCloseSessionPlugins(it.SessionID)
			}
		}
		// Source blank rows have no general session actions. The explicit native
		// plugin entry is the only exception, so it cannot rename/archive a draft.
		if it.Blank {
			return
		}
		keepPin := m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "pin")
		if keepPin && it.Pinned {
			if m.OnUnpin != nil && menu.Item(i18n.T(i18n.KeySidebarUnpin)).Key("native:pin").Icon(sessionPinArtwork).Chosen() {
				m.OnUnpin(it.SessionID)
			}
		} else if keepPin && m.OnPin != nil && menu.Item(i18n.T(i18n.KeySidebarPin)).Key("native:pin").Icon(menuPinArtwork).Chosen() {
			m.OnPin(it.SessionID)
		}
		if keepNative := m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "rename"); keepNative && m.OnBeginRename != nil && menu.Item(i18n.T(i18n.KeySidebarRename)).Key("native:rename").Icon(menuEditArtwork).ShortcutLabel(m.MenuShortcutLabels["session.rename"]).Chosen() {
			m.OnBeginRename(it.SessionID, it.Title)
		}
		if keepNative := m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "fork"); keepNative && m.OnFork != nil && menu.Item(i18n.T(i18n.KeySidebarFork)).Key("native:fork").Icon(menuBranchArtwork).ShortcutLabel(m.MenuShortcutLabels["session.fork"]).Chosen() {
			m.OnFork(it.SessionID)
		}
		if keepNative := m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "archive"); keepNative && m.OnArchive != nil && menu.Item(i18n.T(i18n.KeySidebarArchive)).Key("native:archive").Icon(menuArchiveArtwork).ShortcutLabel(m.MenuShortcutLabels["session.archive"]).Chosen() {
			m.OnArchive(it.SessionID)
		}
		if keepNative := m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "session-id"); keepNative && m.OnCopySessionID != nil && menu.Item(i18n.T(i18n.KeySidebarCopyID)).Key("native:session-id").Icon(menuCopyArtwork).Chosen() {
			m.OnCopySessionID(it.SessionID)
		}
		m.PublicContributions.RenderMenuBefore(menu, "sidebar.workspaces.session.menu.item", it.SessionID, "")
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
func rowArchivedSession(c *ui.Context, id, title string, depth int, m Model, muted ui.Color) {
	m = workspaceOwnerRowModel(m, id)
	if !m.ownerRow && batchRow(c, m, id, title, muted) {
		return
	}
	// Keep the caller's title fallback while recovering real row metadata.
	// A missing summary has no timestamp or path to display in its preview.
	item := session.Item{SessionID: id, Title: title, Blank: true}
	for _, candidate := range m.Snapshot.Items {
		if candidate.SessionID == id {
			item = candidate
			break
		}
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	row := ui.Row(c.Key("archived-session-"+id)).FillWidth().Height(32).Padding(0, 8, 0, float32(8+depth*12)).Gap(0).AlignItems(ui.Center).Radius(12).Disabled(m.ownerRow && m.Batch.Enabled)
	if row.Hovered() || row.FocusWithin() || row.MenuOpen() {
		row.Background(selectedBackground(colors))
	}
	menu := func(menu *ui.Menu) {
		menu.CloseOnPointerLeave(true)
		if m.OnUnarchive != nil && menu.Item(i18n.T(i18n.KeySidebarUnarchive)).Icon(menuArchiveArtwork).Chosen() {
			m.OnUnarchive(id)
		}
		if m.OnDelete != nil && menu.Item(i18n.T(i18n.KeySidebarDelete)).Icon(menuTrashArtwork).Danger(true).Chosen() {
			m.OnDelete(id)
		}
		if m.ownerRow {
			m.PublicContributions.RenderMenu(menu, "sidebar.workspaces.session.menu.item", id)
		}
	}
	row.ContextMenu(menu).Children(func() {
		ui.Box(c).Width(16).Height(20)
		ui.Text(c, title).MinWidth(0).Grow(1).Padding(0, 6, 0, 4).SingleLine().FontSize(theme.InterfaceSize(14)).TextColor(muted)
		if !item.Blank {
			publicSidebarSlot(c, m, "sidebar.session.row.badge", id, nil)
		}
		showActions := row.Hovered() || row.FocusWithin() || row.MenuOpen()
		if !showActions && !item.Blank && item.UpdatedAt > 0 {
			ui.Text(c, sessionTrailingLabel("", item.UpdatedAt)).Label("Session time " + id).
				FontSize(theme.InterfaceSize(10)).FixedLineHeight(16).TextColor(muted).SingleLine()
		}
		more := ui.Button(c.Key("archived-session-menu-"+id), "").Label(sessionActionsLabel(id)).Width(16).Height(16).Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Center().Menu(menu)
		more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(16, 16).TextColor(muted) })
		if !showActions && !more.Focused() {
			more.Absolute().Right(8).Opacity(0)
		}
	})
	publicSidebarRowContext(m, id, row.MenuOpen(), row.Hovered() && !row.MenuOpen() && !m.Batch.Enabled)
	archivedSessionHoverCard(c, m, row, item, title)
}

package sidebar

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func sessionActionsLabel(id, title string) string {
	name := title
	if name == "" {
		name = i18n.T(i18n.KeySidebarNewSessionFallback)
	}
	return fmt.Sprintf(workspaceText("会话“%s”的操作", "Actions for session “%s”"), name)
}

// rowActionGlyph is one hover action: a bare 16px glyph button
// (Rows.module.css .iconButton) that reaches the same callbacks as the row menu.
// Nothing renders without an owner for the action.
func sessionRowAction(c *ui.Context, key string, art *ui.SVG, label, tip string, ink ui.Color, run func()) {
	if run == nil || art == nil {
		return
	}
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonIconXS,
		Label:   label,
		Key:     "session-action-" + key,
		Content: func() {},
	}).Tooltip(tip).Size(design.IconSM, design.IconSM).Padding(0).BorderWidth(0).Center()
	button.Children(func() { ui.Icon(c, art).Size(design.IconXS, design.IconXS).TextColor(ink) })
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
		return i18n.T(i18n.KeyWSMenuUnpinSession)
	}
	return i18n.T(i18n.KeyWSMenuPinSession)
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
		if it.Blank {
			return
		}
		if it.Pinned {
			if m.OnUnpin != nil && menu.Item(i18n.T(i18n.KeySidebarUnpin)).Key("native:pin").Icon(sessionPinArtwork).Chosen() {
				m.OnUnpin(it.SessionID)
			}
		} else if m.OnPin != nil && menu.Item(i18n.T(i18n.KeySidebarPin)).Key("native:pin").Icon(menuPinArtwork).Chosen() {
			m.OnPin(it.SessionID)
		}
		if m.OnBeginRename != nil && menu.Item(i18n.T(i18n.KeySidebarRename)).Key("native:rename").Icon(menuEditArtwork).ShortcutLabel(m.MenuShortcutLabels["session.rename"]).Chosen() {
			m.OnBeginRename(it.SessionID, it.Title)
		}
		if m.OnFork != nil && menu.Item(i18n.T(i18n.KeySidebarFork)).Key("native:fork").Icon(menuBranchArtwork).ShortcutLabel(m.MenuShortcutLabels["session.fork"]).Chosen() {
			m.OnFork(it.SessionID)
		}
		if m.OnArchive != nil && menu.Item(i18n.T(i18n.KeySidebarArchive)).Key("native:archive").Icon(menuArchiveArtwork).ShortcutLabel(m.MenuShortcutLabels["session.archive"]).Chosen() {
			m.OnArchive(it.SessionID)
		}
		if m.OnCopySessionID != nil && menu.Item(i18n.T(i18n.KeySidebarCopyID)).Key("native:session-id").Icon(menuCopyArtwork).Chosen() {
			m.OnCopySessionID(it.SessionID)
		}
	}
}

func sessionStatusSlot(c *ui.Context, status string, muted ui.Color) {
	slot := ui.Box(c).Width(16).Height(20).Label(statusLabel(status))
	if status == "" {
		return
	}
	t := tok(c)
	if status == "running" {
		slot.AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			coss.Spinner(c, coss.SpinnerOptions{Size: design.IconXS, Color: muted})
		})
		return
	}
	ink := t.Warning
	if status == "completed" {
		ink = t.Success
	}
	slot.Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.Circle(r.X+8, r.Y+10, 3)
		p.FillPath(path, ink)
	})
}

// Archived rows are inert: restoring and deleting remain explicit menu actions.
func rowArchivedSession(c *ui.Context, id, title string, depth int, m Model, muted ui.Color) {
	if batchRow(c, m, id, title, muted) {
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
	t := tok(c)
	row := ui.Row(c.Key("archived-session-"+id)).FillWidth().Height(32).Padding(0, design.Sp(2), 0, design.Sp(2)+float32(depth)*design.Sp(3)).Gap(0).AlignItems(ui.Center).Radius(design.RadiusXL)
	if row.Hovered() || row.FocusWithin() || row.MenuOpen() {
		row.Background(sbSelectedBg(t))
	}
	menu := func(menu *ui.Menu) {
		menu.CloseOnPointerLeave(true)
		if m.OnUnarchive != nil && menu.Item(i18n.T(i18n.KeySidebarUnarchive)).Icon(menuArchiveArtwork).Chosen() {
			m.OnUnarchive(id)
		}
		if m.OnDelete != nil && menu.Item(i18n.T(i18n.KeySidebarDelete)).Icon(menuTrashArtwork).Danger(true).Chosen() {
			m.OnDelete(id)
		}
	}
	row.ContextMenu(menu).Children(func() {
		ui.Box(c).Width(16).Height(20)
		ui.Text(c, title).MinWidth(0).Grow(1).Padding(0, design.Sp(1.5), 0, design.Sp(1)).SingleLine().FontSize(fs(14)).TextColor(muted)
		showActions := row.Hovered() || row.FocusWithin() || row.MenuOpen()
		if !showActions && !item.Blank && item.UpdatedAt > 0 {
			ui.Text(c, sessionTrailingLabel("", item.UpdatedAt)).Label("Session time " + id).
				FontSize(fs(10)).FixedLineHeight(16).TextColor(muted).SingleLine()
		}
		more := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonIconXS,
			Label:   sessionActionsLabel(id, title),
			Key:     "archived-session-menu-" + id,
			Content: func() {},
		}).Size(design.IconSM, design.IconSM).Padding(0).BorderWidth(0).TextColor(muted).Center().Menu(menu)
		more.Children(func() { ui.Icon(c, menuEllipsisArtwork).Size(design.IconSM, design.IconSM).TextColor(muted) })
		if !showActions && !more.Focused() {
			more.Absolute().Right(design.Sp(2)).Opacity(0)
		}
	})
	archivedSessionHoverCard(c, m, row, item, title, row.Pressed())
}

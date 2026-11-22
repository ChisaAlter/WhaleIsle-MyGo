package sidebar

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

var batchChecklistIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M3.75 6.25C4.7165 6.25 5.5 5.4665 5.5 4.5C5.5 3.5335 4.7165 2.75 3.75 2.75C2.7835 2.75 2 3.5335 2 4.5C2 5.4665 2.7835 6.25 3.75 6.25Z"/><path d="M7.5 4.5H13.5"/><path d="M3.75 13.25C4.7165 13.25 5.5 12.4665 5.5 11.5C5.5 10.5335 4.7165 9.75 3.75 9.75C2.7835 9.75 2 10.5335 2 11.5C2 12.4665 2.7835 13.25 3.75 13.25Z"/><path d="M7.5 11.5H13.5"/></svg>`))

// VisibleBatchTargets mirrors the rendered tree, including collapsed ancestor
// groups and the dedicated search surface. Browser scope changes clear choices.
func VisibleBatchTargets(m Model) []session.SessionBatchTarget {
	var out []session.SessionBatchTarget
	seen := map[string]bool{}
	knownBlank := map[string]bool{}
	for _, item := range m.Snapshot.Items {
		if item.Blank {
			knownBlank[item.SessionID] = true
		}
	}
	add := func(id, title string) {
		if id != "" && !seen[id] && !knownBlank[id] {
			seen[id] = true
			out = append(out, session.SessionBatchTarget{ID: id, Title: title})
		}
	}
	if m.SearchQuery != nil && strings.TrimSpace(*m.SearchQuery) != "" {
		rows, _ := deriveSessionSearchResults(m.Snapshot, *m.SearchQuery, searchArchiveFilter(m))
		for _, row := range rows {
			add(row.Item.SessionID, row.Item.Title)
		}
		return out
	}
	hidden := map[string]bool{}
	items := sidebarItemsForArchiveFilter(m.Snapshot, m.SearchQuery, browseArchiveFilter(m))
	for _, g := range visibleWorkspaceGroups(m, items) {
		if hidden[g.ParentID] || session.WorkspaceGroupCollapsed(m.Snapshot, g.Key) {
			hidden[g.Key] = true
			continue
		}
		for _, row := range visibleWorkspaceSessionRows(m, g).Rows {
			add(row.Item.SessionID, row.Item.Title)
		}
	}
	if m.archiveExpanded != nil && *m.archiveExpanded {
		for _, item := range archivedSectionItems(m) {
			title := item.Title
			if title == "" {
				title = item.SessionID
			}
			add(item.SessionID, title)
		}
	}
	return out
}

// batchRow also serves archived summary-only rows: lack of a summary is not
// evidence of a blank session. Only an actual Blank flag suppresses selection.
func batchRow(c *ui.Context, m Model, id, title string, fg ui.Color) bool {
	if !m.Batch.Enabled {
		return false
	}
	item := session.Item{SessionID: id, Title: title}
	for _, candidate := range m.Snapshot.Items {
		if candidate.SessionID == id {
			item = candidate
			break
		}
	}
	return batchSessionRow(c, m, item, 0, title, fg, fg)
}

func batchSessionRow(c *ui.Context, m Model, item session.Item, depth int, title string, fg, muted ui.Color) bool {
	if !m.Batch.Enabled {
		return false
	}
	if item.Blank {
		title = i18n.T(i18n.KeySidebarNewSessionFallback)
	}
	if strings.TrimSpace(title) == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	locked := m.Batch.Busy || m.Batch.Confirming || m.OnBatchEdit == nil
	row := ui.Row(c.Key("batch-row-"+item.SessionID)).Label("Select session "+item.SessionID).FillWidth().Height(32).Radius(12).
		Background(ui.Transparent).BorderWidth(0).Padding(0, 8, 0, float32(8+depth*12)).Clip().AlignItems(ui.Center).Gap(0)
	selected, changed := false, false
	_, selected = m.Batch.Selected[item.SessionID]
	if item.Blank {
		row.Label(title)
	} else {
		row.Disabled(locked)
		if !isArchived(m.Snapshot, item.SessionID) {
			row.Focusable()
		}
	}
	if selected {
		row.Background(c.Theme().Accent.Alpha(.12))
	}
	row.Children(func() {
		if item.Blank {
			ui.Box(c).Size(16, 20)
		} else {
			ui.Box(c).Size(16, 20).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				check := ui.Checkbox(c.Key("batch-checkbox-"+item.SessionID), &selected, "").Label(i18n.WSf(i18n.KeyWSSelectionSessionAria, "name", title)).Size(16, 16).Gap(0).Disabled(locked)
				changed = check.Changed()
			})
		}
		ui.Text(c, title).MinWidth(0).Grow(1).Padding(0, 6, 0, 4).SingleLine().FontSize(theme.InterfaceSize(14)).TextColor(fg)
		if !item.Blank {
			ui.Text(c, sessionTrailingLabel(sessionStatus(m, item), item.UpdatedAt)).Label("Session time " + item.SessionID).FontSize(theme.InterfaceSize(10)).FixedLineHeight(16).SingleLine().TextColor(muted)
		}
	})
	clicked := false
	if !item.Blank && !isArchived(m.Snapshot, item.SessionID) {
		clicked = row.Clicked()
	}
	if !item.Blank && !locked && (changed || clicked) {
		m.OnBatchEdit("select", []session.SessionBatchTarget{{ID: item.SessionID, Title: title}})
	}
	return true
}
func batchToggle(c *ui.Context, m Model, fg ui.Color) {
	if m.OnBatchEdit == nil {
		return
	}
	b := m.Batch
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	label := i18n.T(i18n.KeyWSSelectionEnter)
	if b.Enabled {
		label = i18n.T(i18n.KeyWSSelectionExit)
	}
	button := ui.Button(c, "").Label(label).Tooltip(label).Width(28).Height(28).Padding(0).Radius(8).
		BorderWidth(0).Background(ui.Transparent).Disabled(b.Busy || b.Confirming).Center()
	ink := fg
	if b.Enabled || button.Hovered() {
		button.Background(chrome.Color(colors.Hover))
		ink = chrome.Color(colors.FG)
	}
	button.Children(func() {
		ui.Icon(c, batchChecklistIcon).Size(14, 14).TextColor(ink)
	})
	if button.Clicked() {
		m.OnBatchEdit("toggle", nil)
	}
}

// batchActionIcon draws the 14 DIP leading glyph a SessionSelectionToolbar
// action carries before its label (SessionSelection.tsx renders size-14 icons).
func batchActionIcon(icon *ui.SVG, ink ui.Color) func(*ui.Context) {
	return func(c *ui.Context) { ui.Icon(c, icon).Size(14, 14).TextColor(ink) }
}

func batchToolbar(c *ui.Context, m Model, fg ui.Color) {
	if m.OnBatchEdit == nil {
		return
	}
	b := m.Batch
	locked := b.Busy || b.Confirming
	if !b.Enabled {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	// Progress shares the selection count's caption seat, so a running batch
	// does not push the session list down.
	count := i18n.WSf(i18n.KeyWSSelectionCount, "n", strconv.Itoa(len(b.Selected)))
	if b.Busy {
		count = fmt.Sprintf(workspaceText("处理中 %d / %d", "Processing %d / %d"), b.Attempted, len(b.Targets))
	}
	none := len(b.Selected) == 0
	dangerInk := chrome.Color(colors.Danger)
	// --dsw-alias-interactive-bg-hover-danger is error-primary at 5% light and
	// 15% dark (ui-theme design-platform.css); SessionSelection's .danger is a
	// label-only tint, so the outline stays border-l3.
	dangerHover := dangerInk.Alpha(.05)
	if m.SystemDark {
		dangerHover = dangerInk.Alpha(.15)
	}
	// SessionSelection.module.css .toolbar: an 8 DIP column under 4/4/12
	// padding; .selectionLine carries the label-primary count and two ghost sm
	// buttons with padding-inline 8, and .actions shares the row between two
	// outline sm buttons at flex 1.
	ui.Column(c).FillWidth().Gap(8).Padding(4, 4, 12, 4).Children(func() {
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Text(c, count).Grow(1).MinWidth(0).SingleLine().Padding(0, 0, 0, 4).FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(fg).Label(i18n.T(i18n.KeyWSSelectionToolbarAria))
			if widgets.Button(c, colors, i18n.T(i18n.KeyWSSelectionAllVisible), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: locked, PadX: 8}).Clicked() {
				m.OnBatchEdit("visible", VisibleBatchTargets(m))
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyWSSelectionClear), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: locked || none, PadX: 8}).Clicked() {
				m.OnBatchEdit("clear", nil)
			}
		})
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			if widgets.Button(c, colors, i18n.T(i18n.KeyWSSelectionArchive), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: locked || none, Leading: batchActionIcon(menuArchiveArtwork, fg)}).Grow(1).MinWidth(0).Clicked() && m.OnBatchRequest != nil {
				m.OnBatchRequest("archive")
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyWSSelectionDelete), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: locked || none, Ink: &dangerInk, HoverFill: &dangerHover, Leading: batchActionIcon(menuTrashArtwork, dangerInk)}).Grow(1).MinWidth(0).Clicked() && m.OnBatchRequest != nil {
				m.OnBatchRequest("delete")
			}
		})
	})
}

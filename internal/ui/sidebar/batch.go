package sidebar

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
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

func batchCheckboxSlot(c *ui.Context, checked *bool, disabled bool, label, key string) bool {
	t := tok(c)
	changed := false
	slot := ui.Box(c).Size(design.IconSM, design.Sp(5)).AlignItems(ui.Center).Justify(ui.Center)
	slot.Draw(func(p *ui.Painter, r ui.Rect) {
		if *checked {
			return
		}
		box := ui.Rect{X: r.X + (r.W-16)/2, Y: r.Y + (r.H-16)/2, W: 16, H: 16}
		bg := ui.RGBA(0, 0, 0, 0.04)
		border := ui.RGBA(0, 0, 0, 0.30)
		if t.IsDark() {
			bg = ui.RGBA(255, 255, 255, 0.08)
			border = ui.RGBA(255, 255, 255, 0.38)
		}
		p.Fill(box, bg, 4)
		p.Stroke(box, border, 4, 1)
	})
	slot.Children(func() {
		check := coss.Checkbox(c, checked, "", coss.CheckOptions{
			Disabled: disabled,
			Label:    label,
			Key:      key,
		})
		changed = check.Changed()
	})
	return changed
}

func batchSessionRow(c *ui.Context, m Model, item session.Item, depth int, title string, fg, muted ui.Color) bool {
	if !m.Batch.Enabled {
		return false
	}
	t := tok(c)
	if item.Blank {
		title = i18n.T(i18n.KeySidebarNewSessionFallback)
	}
	if strings.TrimSpace(title) == "" {
		title = i18n.T(i18n.KeyWSSessionUntitled)
	}
	locked := m.Batch.Busy || m.Batch.Confirming || m.OnBatchEdit == nil
	row := ui.Row(c.Key("batch-row-"+item.SessionID)).Label("Select session "+item.SessionID).FillWidth().Height(design.Sp(8)).Radius(design.RadiusXL).
		Background(ui.Transparent).BorderWidth(0).Padding(0, design.Sp(2), 0, design.Sp(2)+float32(depth)*design.Sp(3)).Clip().AlignItems(ui.Center).Gap(0)
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
		row.Background(sbSelectedBg(t))
	} else if !item.Blank && (row.Hovered() || row.FocusWithin()) {
		row.Background(sbHoverBg(t))
	}
	row.Children(func() {
		if item.Blank {
			ui.Box(c).Size(design.IconSM, design.Sp(5))
		} else {
			changed = batchCheckboxSlot(c, &selected, locked, i18n.WSf(i18n.KeyWSSelectionSessionAria, "name", title), "batch-checkbox-"+item.SessionID)
		}
		ui.Text(c, title).MinWidth(0).Grow(1).Padding(0, design.Sp(1.5), 0, design.Sp(1)).SingleLine().FontSize(fs(14)).TextColor(sbRowInk(t, selected, fg))
		if !item.Blank {
			ui.Text(c, sessionTrailingLabel(sessionStatus(m, item), item.UpdatedAt)).Label("Session time " + item.SessionID).FontSize(fs(10)).FixedLineHeight(design.Sp(4)).SingleLine().TextColor(muted)
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
	t := tok(c)
	b := m.Batch
	label := i18n.T(i18n.KeyWSSelectionEnter)
	if b.Enabled {
		label = i18n.T(i18n.KeyWSSelectionExit)
	}
	btn := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconSM,
		Disabled: b.Busy || b.Confirming,
		Pressed:  b.Enabled,
		Label:    label,
		Key:      "batch-toggle",
		Content:  func() {},
	}).Tooltip(label).Radius(design.RadiusMD).Padding(0).BorderWidth(0).Center()
	ink := fg
	if b.Enabled || btn.Hovered() {
		ink = t.Foreground
	}
	btn.Children(func() {
		ui.Icon(c, batchChecklistIcon).Size(design.IconXS, design.IconXS).TextColor(ink)
	})
	if btn.Clicked() {
		m.OnBatchEdit("toggle", nil)
	}
}

func batchActionContent(c *ui.Context, icon *ui.SVG, label string, ink ui.Color) func() {
	return func() {
		ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Center().Children(func() {
			ui.Icon(c, icon).Size(design.IconXS, design.IconXS).TextColor(ink)
			ui.Text(c, label).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).TextColor(ink).SingleLine()
		})
	}
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
	t := tok(c)
	count := i18n.WSf(i18n.KeyWSSelectionCount, "n", strconv.Itoa(len(b.Selected)))
	if b.Busy {
		count = fmt.Sprintf(workspaceText("处理中 %d / %d", "Processing %d / %d"), b.Attempted, len(b.Targets))
	}
	none := len(b.Selected) == 0
	ui.Column(c).FillWidth().Gap(design.Sp(2)).Padding(design.Sp(1), design.Sp(1), design.Sp(3), design.Sp(1)).Children(func() {
		ui.Row(c).FillWidth().Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, count).Grow(1).MinWidth(0).SingleLine().Padding(0, 0, 0, design.Sp(1)).FontSize(fs(12)).FixedLineHeight(18).TextColor(fg).Label(i18n.T(i18n.KeyWSSelectionToolbarAria))
			if coss.Button(c, i18n.T(i18n.KeyWSSelectionAllVisible), coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonSM,
				Disabled: locked,
				Key:      "batch-all-visible",
			}).BorderWidth(0).Padding(0, design.Sp(2)).Clicked() {
				m.OnBatchEdit("visible", VisibleBatchTargets(m))
			}
			if coss.Button(c, i18n.T(i18n.KeyWSSelectionClear), coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonSM,
				Disabled: locked || none,
				Key:      "batch-clear",
			}).BorderWidth(0).Padding(0, design.Sp(2)).Clicked() {
				m.OnBatchEdit("clear", nil)
			}
		})
		ui.Row(c).FillWidth().Gap(design.Sp(2)).Children(func() {
			archiveLabel := i18n.T(i18n.KeyWSSelectionArchive)
			archiveDisabled := locked || none
			archiveInk := t.Foreground
			archiveBtn := coss.Button(c, archiveLabel, coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonSM,
				Disabled: archiveDisabled,
				Content:  batchActionContent(c, menuArchiveArtwork, archiveLabel, archiveInk),
				Key:      "batch-archive",
			}).Label(archiveLabel).Grow(1).MinWidth(0)
			if t.IsDark() {
				bg := ui.RGBA(255, 255, 255, 0.10)
				if !archiveDisabled && archiveBtn.Hovered() {
					bg = ui.RGBA(255, 255, 255, 0.16)
				}
				archiveBtn.Background(bg).Border(design.BorderWidth, ui.RGBA(255, 255, 255, 0.22))
			}
			if archiveBtn.Clicked() && m.OnBatchRequest != nil {
				m.OnBatchRequest("archive")
			}
			deleteLabel := i18n.T(i18n.KeyWSSelectionDelete)
			deleteDisabled := locked || none
			deleteInk := t.DestructiveForeground
			if t.IsDark() {
				deleteInk = design.TW.Red300
			}
			deleteBtn := coss.Button(c, deleteLabel, coss.ButtonOptions{
				Variant:  coss.ButtonDestructiveOutline,
				Size:     coss.ButtonSM,
				Disabled: deleteDisabled,
				Content:  batchActionContent(c, menuTrashArtwork, deleteLabel, deleteInk),
				Key:      "batch-delete",
			}).Label(deleteLabel).Grow(1).MinWidth(0)
			if t.IsDark() {
				bg := t.Destructive.Alpha(0.18)
				if !deleteDisabled && deleteBtn.Hovered() {
					bg = t.Destructive.Alpha(0.28)
				}
				deleteBtn.Background(bg).Border(design.BorderWidth, t.Destructive.Alpha(0.50))
			}
			if deleteBtn.Clicked() && m.OnBatchRequest != nil {
				m.OnBatchRequest("delete")
			}
		})
	})
}

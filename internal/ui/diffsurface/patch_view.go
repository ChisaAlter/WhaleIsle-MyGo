package diffsurface

import (
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/diff"
)

// diffToolButton paints one compact control of the patch tool row.
func diffToolButton(c *ui.Context, label string, selected bool, onClick func()) ui.Element {
	button := ui.Button(c, label).Height(26).Padding(0, 10).Radius(6).BorderWidth(0).FontSize(fonttheme.InterfaceSize(12)).Selected(selected).Tooltip(label)
	if selected {
		button.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	} else {
		button.Background(ui.Transparent).TextColor(c.Theme().TextMuted)
	}
	if button.Hovered() && !selected {
		button.Background(ui.RGBA(c.Theme().TextMuted.R, c.Theme().TextMuted.G, c.Theme().TextMuted.B, .14))
	}
	if button.Clicked() && onClick != nil {
		onClick()
	}
	return button
}

// paintDiffControls paints the tool row above the patch: the view mode as a
// two-chip choice, then wrap and fold controls.
func paintDiffControls(c *ui.Context, m Model) {
	if m.Workbench == nil {
		return
	}
	view := &m.Workbench.View
	source := ""
	if m.Text != nil {
		source = *m.Text
	}
	view.update(source)
	ui.Row(c).FillWidth().Gap(4).Padding(6, 12, 6, 12).Wrap().AlignItems(ui.Center).Children(func() {
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffUnified), !view.Split, func() { view.Split = false })
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffSplit), view.Split, func() { view.Split = true })
		ui.Box(c).Width(0.5).Height(16).Shrink(0).Background(c.Theme().Border)
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffWrap), view.Wrap, func() { view.Wrap = !view.Wrap })
		ui.Box(c).Grow(1)
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffExpandAll), false, func() { view.setAllClosed(false); view.setAllHunks(false) })
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffCollapseAll), false, func() { view.setAllClosed(true); view.setAllHunks(true) })
	})
}

// paintPatchFiles paints every file of a patch as a collapsible block: its path
// header, then the unified or split body.
func paintPatchFiles(c *ui.Context, m Model, source string, st diff.Style) {
	if m.Workbench == nil {
		paintUnifiedStandalone(c, source, st)
		return
	}
	view := &m.Workbench.View
	for _, file := range view.update(source) {
		f := file
		ui.Column(c).FillWidth().MinWidth(0).Gap(3).Padding(0, 0, 6, 0).Children(func() {
			open := !view.Closed[f.Path]
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				title := f.Path
				if title == "" {
					title = word("差异内容", "Diff contents")
				}
				head := ui.Button(c, title).FontSize(fonttheme.InterfaceSize(12)).Grow(1).MinWidth(0).Height(28).Radius(6).Padding(0, 8).BorderWidth(0).Background(ui.Transparent).TextColor(st.FG).Disabled(m.OnOpenFile == nil || !safePatchPath(f.Path)).Tooltip(f.Path)
				if head.Hovered() && m.OnOpenFile != nil && safePatchPath(f.Path) {
					head.Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .14))
				}
				if head.Clicked() && m.OnOpenFile != nil {
					if err := m.OnOpenFile(f.Path); err != nil {
						m.Workbench.Status = err.Error()
					}
				}
				paintPatchFileActions(c, m, f.Path)
				fold := i18n.T(i18n.KeySurfacesDiffExpandAll)
				if open {
					fold = i18n.T(i18n.KeySurfacesDiffCollapseAll)
				}
				toggle := ui.Button(c, "").Label(fold).Size(22, 22).Shrink(0).Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(fold)
				toggle.Children(func() {
					if open {
						ui.Icon(c, diffChevronDownIcon).Size(10, 10).TextColor(st.Muted).PassThrough()
					} else {
						ui.Icon(c, diffChevronRightIcon).Size(10, 10).TextColor(st.Muted).PassThrough()
					}
				})
				if toggle.Hovered() {
					toggle.Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .14))
				}
				if toggle.Clicked() {
					view.Closed[f.Path] = open
				}
			})
			if !open {
				return
			}
			render := func() {
				fold := hunkFold{collapsed: view.foldedHunks(f.Path), onToggle: func(index int) { view.toggleHunk(f.Path, index) }}
				paintUnifiedRows(c, f.Rows, st, view.Wrap, fold)
			}
			renderSplit := func() {
				fold := hunkFold{collapsed: view.foldedHunks(f.Path), onToggle: func(index int) { view.toggleHunk(f.Path, index) }}
				paintSplitRows(c, f.Rows, st, view.Wrap, fold)
			}
			if view.Split {
				// The two-column view scrolls as one block so both sides stay in
				// step; the unified view owns its scroller to keep the gutter put.
				if view.Wrap {
					renderSplit()
				} else {
					ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(renderSplit)
				}
			} else {
				render()
			}
		})
	}
}

// Staging belongs to the file being reviewed, not a context-only action on a
// separate commit checklist. The commit dialog keeps its own explicit selection.
func paintPatchFileActions(c *ui.Context, m Model, path string) {
	w := m.Workbench
	if w == nil || w.BaseRef != "" || !safePatchPath(path) {
		return
	}
	for _, change := range w.Changes {
		if change.Path != path {
			continue
		}
		if change.Unstaged() && ui.Button(c, word("暂存", "Stage")).Height(24).Shrink(0).Padding(0, 6).FontSize(fonttheme.InterfaceSize(12)).Disabled(w.Operation != nil).Clicked() {
			stageCommitFile(m, change)
		}
		if change.Staged() && ui.Button(c, word("取消暂存", "Unstage")).Height(24).Shrink(0).Padding(0, 6).FontSize(fonttheme.InterfaceSize(12)).Disabled(w.Operation != nil).Clicked() {
			unstageCommitFile(m, change)
		}
		if change.Unstaged() {
			ui.MenuButton(c, word("更多", "More"), func(menu *ui.Menu) {
				if menu.Item(word("丢弃此文件…", "Discard file…")).Disabled(w.Operation != nil).Chosen() {
					reviewDiscard(m, path)
				}
			}).Height(24).Shrink(0).FontSize(fonttheme.InterfaceSize(12))
		}
		return
	}
}

// paintSplitRows paints the two-column view: the before cell on the left and the
// after cell on the right, each tinted by its own change kind.
func paintSplitRows(c *ui.Context, rows []patchRow, st diff.Style, wrap bool, fold hunkFold) {
	minWidth := float32(0)
	if !wrap {
		minWidth = splitColumnWidth(rows)
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(1).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Padding(0, 8, 4, 8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeySurfacesDiffBefore)).Grow(1).MinWidth(minWidth).FontSize(fonttheme.InterfaceSize(11)).TextColor(st.Muted)
			ui.Text(c, i18n.T(i18n.KeySurfacesDiffAfter)).Grow(1).MinWidth(minWidth).FontSize(fonttheme.InterfaceSize(11)).TextColor(st.Muted)
		})
		hunk, hidden := -1, false
		for _, row := range rows {
			r := row
			if r.Meta != "" {
				if len(r.Meta) > 1 && r.Meta[0] == '@' && r.Meta[1] == '@' {
					hunk++
					hidden = fold.collapsed[hunk]
					paintHunkHeader(c, r.Meta, hunk, st, fold)
					continue
				}
				paintDiffLine(c, diffLine{Kind: diffLineMeta, Text: r.Meta}, st, minWidth, wrap)
				continue
			}
			if hidden {
				continue
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				paintSplitCell(c, r.Old, r.OldNumber, wrap, minWidth, st)
				paintSplitCell(c, r.New, r.NewNumber, wrap, minWidth, st)
			})
		}
	})
}

// paintUnifiedStandalone paints a patch when no workbench owns the pane; the
// panel then has no per-file folding, so the whole patch is one column.
func paintUnifiedStandalone(c *ui.Context, source string, st diff.Style) {
	rows := splitPatchRows(source)
	if len(rows) == 0 {
		return
	}
	ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
		paintUnifiedRows(c, rows, st, false, hunkFold{})
	})
}

// splitColumnWidth estimates the pixel width of the widest line in a hunk.
func splitColumnWidth(rows []patchRow) float32 {
	longest := 20
	for _, r := range rows {
		for _, line := range []string{r.Old, r.New} {
			if n := len([]rune(line)); n > longest {
				longest = n
			}
		}
	}
	return float32(longest+7) * 7.5
}

package diffsurface

import (
	"fmt"
	"strconv"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

// commitDialogWidth is the reviewed commit form's width: narrow enough to read
// as a focused task, wide enough for a path and its status.
const commitDialogWidth float32 = 520

// The checklist's line counts use the diff renderer's palette, so one glance
// reads the same green and red in the list and in the patch below it.
var (
	commitAddedColor   = ui.RGB(61, 153, 112)
	commitDeletedColor = ui.RGB(224, 102, 102)
)

// commitDialog paints the commit flow as one in-app form: the file selection,
// the message, the branch and PR options and the review result all live in the
// dialog, so the diff pane keeps only the workbench list and one launcher.
func commitDialog(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil || !w.CommitOpen {
		return
	}
	open := true
	ui.DialogBase(c.Key("commit-dialog"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), commitDialogWidth, 20).
			MaxHeightPercent(88).Gap(12).
			Label(i18n.T(i18n.KeySurfacesGitCommitTitle)).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeySurfacesGitCommitTitle)).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(16)).FontWeight(600).TextColor(fg)
				if ui.Button(c, "").Label(i18n.T(i18n.KeySurfacesGitClose)).Size(24, 24).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Clicked() {
					open = false
				}
			})
			branch := word("detached HEAD", "detached HEAD")
			if w.RepositoryStatus != nil && w.RepositoryStatus.Branch != "" {
				branch = w.RepositoryStatus.Branch
			}
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitSubtitle), branch, len(w.Changes))).
				FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Scroll(c).FillWidth().MinHeight(0).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					if paintCommitPreparation(c, m, fg, muted) {
						return
					}
					paintCommitFileList(c, m, fg, muted)
					paintCommitSelection(c, m, fg, muted)
					if w.ReviewText != "" {
						ui.ScrollBoth(c).FillWidth().MinWidth(0).Height(140).Children(func() {
							ui.Text(c, w.ReviewText).NoWrap().Selectable().Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
						})
					}
					if w.Status != "" {
						ui.Text(c, w.Status).FillWidth().Role(ui.RoleStatus).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					}
				})
			})
		})
	})
	if !open {
		w.CommitOpen = false
	}
}

// paintCommitFileList is the dialog's staged checklist: one row per change with
// its selection box, its status letters and its path.
func paintCommitFileList(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w == nil {
		return
	}
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, i18n.T(i18n.KeySurfacesGitCommitFiles)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(500).TextColor(fg).Grow(1)
		if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesGitSelectAll), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			selectAllCommitFiles(w, true)
		}
		if widgets.Button(c, colors, i18n.T(i18n.KeySurfacesGitSelectNone), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			selectAllCommitFiles(w, false)
		}
	})
	if len(w.Changes) == 0 {
		ui.Text(c, i18n.T(i18n.KeySurfacesGitCommitEmpty)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
		paintCommitChangeRows(c, m, fg, muted)
	})
}

// paintCommitChangeRows paints the change rows shared by the pane's workbench
// list and the dialog's checklist. The row is a target for the diff and carries
// its staging actions in the context menu, so it does not grow a strip of inline
// buttons.
func paintCommitChangeRows(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil {
		return
	}
	for _, change := range w.Changes {
		ch := change
		row := ui.Row(c.Key("commit-row-"+ch.Path)).FillWidth().MinWidth(0).Gap(8).Height(24).AlignItems(ui.Center)
		if row.Hovered() {
			row.Background(c.Theme().SurfaceHover)
		}
		row.Children(func() {
			paintCommitCheckbox(c, w, ch.Path)
			ui.Text(c, string([]byte{ch.Index, ch.Worktree})).Width(22).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
			ui.Text(c, ch.Path).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).SingleLine().Ellipsis("…").Tooltip(ch.Path)
			if ch.Added > 0 {
				ui.Text(c, "+"+strconv.Itoa(ch.Added)).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(commitAddedColor)
			}
			if ch.Deleted > 0 {
				ui.Text(c, "-"+strconv.Itoa(ch.Deleted)).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(commitDeletedColor)
			}
		})
		row.ContextMenu(func(menu *ui.Menu) {
			if menu.Item(i18n.T(i18n.KeySurfacesGitOpenDiff)).Chosen() {
				scopeDiff(m, ch.Path, !ch.Unstaged())
			}
			if ch.Unstaged() {
				if menu.Item(word("暂存此文件", "Stage file")).Chosen() {
					stageCommitFile(m, ch)
				}
				if menu.Item(word("丢弃此文件…", "Discard file…")).Chosen() {
					reviewDiscard(m, ch.Path)
				}
			}
			if ch.Staged() {
				if menu.Item(word("查看已暂存", "Staged diff")).Chosen() {
					scopeDiff(m, ch.Path, true)
				}
				if menu.Item(word("取消暂存", "Unstage")).Chosen() {
					unstageCommitFile(m, ch)
				}
			}
		})
	}
}

// selectAllCommitFiles stages the whole change set for this commit or clears it.
func selectAllCommitFiles(w *Workbench, selected bool) {
	s := w.commitState()
	if selected {
		s.Excluded = map[string]bool{}
	} else {
		for _, ch := range w.Changes {
			s.Excluded[ch.Path] = true
		}
	}
	s.LargeAcknowledged = false
	w.invalidateCommitReview()
}

// stageCommitFile stages one change and refreshes the workbench.
func stageCommitFile(m Model, ch git.Change) {
	paths := []string{ch.Path}
	if ch.Original != "" {
		paths = append(paths, ch.Original)
	}
	m.Workbench.invalidateCommitReview()
	if _, err := git.StagePaths(m.WorkspaceRoot, paths...); err != nil {
		m.Workbench.Status = err.Error()
	} else {
		m.Workbench.Status = word("已暂存：", "Staged: ") + ch.Path
	}
	m.Workbench.reload(m.WorkspaceRoot)
	refresh(m)
}

// unstageCommitFile clears one change from the index and refreshes the workbench.
func unstageCommitFile(m Model, ch git.Change) {
	paths := []string{ch.Path}
	if ch.Original != "" {
		paths = append(paths, ch.Original)
	}
	m.Workbench.invalidateCommitReview()
	if _, err := git.UnstagePaths(m.WorkspaceRoot, paths...); err != nil {
		m.Workbench.Status = err.Error()
	} else {
		m.Workbench.Status = word("已取消暂存：", "Unstaged: ") + ch.Path
	}
	m.Workbench.reload(m.WorkspaceRoot)
	refresh(m)
}

// commitLauncher is the pane's only commit affordance: the form opens in the
// dialog instead of growing a column of inline buttons.
func commitLauncher(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil {
		return
	}
	label := fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitOpen), len(w.Changes))
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.PrimaryButton(c, label).Disabled(len(w.Changes) == 0 || w.Operation != nil).Clicked() {
			w.CommitOpen = true
		}
		if w.Status != "" {
			ui.Text(c, w.Status).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).SingleLine().Ellipsis("…")
		}
	})
}

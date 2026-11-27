package diffsurface

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

// commitDialogWidth is the reviewed commit form's width: narrow enough to read
// as a focused task, wide enough for a path and its status.
const commitDialogWidth float32 = 520

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
				closeLabel := i18n.T(i18n.KeySurfacesGitClose)
				if coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconX,
					Label:   closeLabel,
					Key:     "commit-dialog-close",
				}).Shrink(0).Tooltip(closeLabel).Clicked() {
					open = false
				}
			})
			branch := word("detached HEAD", "detached HEAD")
			if w.RepositoryStatus != nil && w.RepositoryStatus.Branch != "" {
				branch = w.RepositoryStatus.Branch
			}
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitSubtitle), branch, len(w.Changes))).
				FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			ui.Text(c, GitLocale("commit.description")).
				FillWidth().FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
	w := m.Workbench
	if w == nil {
		return
	}
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, i18n.T(i18n.KeySurfacesGitCommitFiles)).FontSize(fonttheme.InterfaceSize(12)).FontWeight(500).TextColor(fg).Grow(1)
		ui.Text(c, formatSelectedCount(len(w.selectedCommitPaths()), len(w.Changes))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if coss.Button(c, i18n.T(i18n.KeySurfacesGitSelectAll), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "commit-select-all"}).Shrink(0).Clicked() {
			selectAllCommitFiles(w, true)
		}
		if coss.Button(c, i18n.T(i18n.KeySurfacesGitSelectNone), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Key: "commit-select-none"}).Shrink(0).Clicked() {
			selectAllCommitFiles(w, false)
		}
	})
	if len(w.Changes) == 0 {
		if !w.Loaded && w.isReloading(m.WorkspaceRoot) {
			ui.Text(c, word("正在加载仓库变更…", "Loading repository changes…")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		ui.Text(c, i18n.T(i18n.KeySurfacesGitCommitEmpty)).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(2).Padding(4).Radius(8).Border(0.5, c.Theme().Border).Children(func() {
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
	tokens := design.For(c)
	addedColor := tokens.SuccessForeground
	deletedColor := tokens.DestructiveForeground
	s := w.commitState()
	for _, change := range w.Changes {
		ch := change
		selected := !s.Excluded[ch.Path]
		row := ui.Row(c.Key("commit-row-"+ch.Path)).FillWidth().MinWidth(0).Gap(8).Height(28).Padding(0, 8).Radius(6).AlignItems(ui.Center)
		if row.Hovered() {
			row.Background(c.Theme().SurfaceHover)
		}
		row.Children(func() {
			paintCommitCheckbox(c, w, ch.Path)
			toggle := ui.ButtonBase(c.Key("commit-row-toggle-" + ch.Path)).Grow(1).MinWidth(0).FillHeight().Gap(design.Sp(2)).Radius(design.RadiusSM).
				Justify(ui.Start).AlignItems(ui.Center).Cursor(ui.CursorPointer).Role(ui.RoleCheckBox).Checked(selected).Label(ch.Path).Tooltip(ch.Path)
			toggle.Children(func() {
				ui.Text(c, string([]byte{ch.Index, ch.Worktree})).Width(22).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
				ui.Text(c, ch.Path).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).SingleLine().Ellipsis("…")
				if ch.Added > 0 {
					ui.Text(c, "+"+strconv.Itoa(ch.Added)).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(addedColor)
				}
				if ch.Deleted > 0 {
					ui.Text(c, "-"+strconv.Itoa(ch.Deleted)).Shrink(0).Font("monospace").FontSize(fonttheme.InterfaceSize(11)).TextColor(deletedColor)
				}
			})
			if toggle.Clicked() {
				s.Excluded[ch.Path] = selected
				s.LargeAcknowledged = false
				w.invalidateCommitReview()
			}
		})
		row.ContextMenu(func(menu *ui.Menu) {
			if menu.Item(i18n.T(i18n.KeySurfacesGitOpenDiff)).Chosen() {
				scopeDiff(m, ch.Path, !ch.Unstaged())
			}
			if ch.Unstaged() {
				if menu.Item(word("暂存此文件", "Stage file")).Disabled(w.Operation != nil).Chosen() {
					stageCommitFile(m, ch)
				}
				if menu.Item(word("还原…", "Discard file…")).Disabled(w.Operation != nil).Chosen() {
					reviewDiscard(m, ch.Path)
				}
			}
			if ch.Staged() {
				if menu.Item(word("查看已暂存", "Staged diff")).Chosen() {
					scopeDiff(m, ch.Path, true)
				}
				if menu.Item(word("取消暂存", "Unstage")).Disabled(w.Operation != nil).Chosen() {
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
	if m.Workbench == nil {
		return
	}
	m.Workbench.requestStage(m, ch, true)
}

// unstageCommitFile clears one change from the index and refreshes the workbench.
func unstageCommitFile(m Model, ch git.Change) {
	if m.Workbench == nil {
		return
	}
	m.Workbench.requestStage(m, ch, false)
}

// commitLauncher is the pane's only commit affordance: the form opens in the
// dialog instead of growing a column of inline buttons.
func commitLauncher(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil {
		return
	}
	label := fmt.Sprintf(i18n.T(i18n.KeySurfacesGitCommitOpen), len(w.Changes))
	ui.Row(c).FillWidth().Gap(8).Padding(8, 10).Radius(8).Border(0.5, c.Theme().Border).AlignItems(ui.Center).Children(func() {
		if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: len(w.Changes) == 0 || w.Operation != nil, Key: "commit-open"}).Shrink(0).Clicked() {
			w.CommitOpen = true
		}
		if w.Status != "" {
			ui.Text(c, w.Status).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).SingleLine().Ellipsis("…")
		}
	})
}

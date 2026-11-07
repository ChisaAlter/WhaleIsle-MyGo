package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Workbench is window-owned Git UI state. No operation stages or pushes implicitly.
type Workbench struct {
	captionReview *captionReviewState
	Progress      *git.Progress
	LastStack     *git.StackResult
	Remote        *remoteWorkflowState
	Commit        *commitSelectionState
	// CommitOpen mounts the commit form as an in-app dialog; the pane keeps only
	// the launcher so the diff stays readable.
	CommitOpen                                bool
	View                                      patchViewState
	Discard                                   *git.DiscardPlan
	Root                                      string
	BaseRef, CompareQuery, CompareError       string
	CompareOpen                               bool
	CompareBranches                           []git.Branch
	comparison                                *comparisonRequest
	invalidate                                func()
	Changes                                   []git.Change
	Staged                                    bool
	AgainstHEAD                               bool
	Message, Status, ReviewToken, ReviewText  string
	Loaded                                    bool
	Branches                                  []git.Branch
	branchesRequest                           *branchListRequest
	BranchesOpen                              bool
	RepositoryToolsOpen                       bool
	BranchQuery, BranchName, BranchStatus     string
	Repository                                bool
	RepositoryStatus                          *git.RepositoryStatus
	RepositoryStatusError                     string
	repositoryStatusRequest                   *repositoryStatusRequest
	Operation                                 *gitOperation
	PublishOpen, PublishReview, PublishPublic bool
	PublishName, PublishRemoteURL             string
	ReviewedPublish                           git.PublishInput
	PushAfterCommit                           bool
	ReviewedPush                              git.PushPlan
}

func word(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}
func (w *Workbench) reload(root string) {
	w.Root = root
	w.Loaded = true
	w.reloadRepositoryStatus()
	w.Repository = git.IsRepository(root)
	changes, err := git.Changes(root)
	if err != nil {
		w.Status = err.Error()
		w.Changes = nil
		return
	}
	w.Changes = changes
	w.scanCommitFiles()
}
func refreshWorkbench(m Model) {
	if m.Workbench != nil {
		w := m.Workbench
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		if w.Loaded && w.Root != m.WorkspaceRoot {
			w.cancelBranchList()
			w.cancelRepositoryStatus()
			w.CancelComparison()
			operation := w.Operation
			*w = Workbench{Operation: operation, invalidate: w.invalidate, AgainstHEAD: true}
			if m.PathFilter != nil {
				*m.PathFilter = ""
			}
		}
		w.reload(m.WorkspaceRoot)
	}
	refresh(m)
}
func scopeDiff(m Model, path string, staged bool) {
	if m.PathFilter != nil {
		*m.PathFilter = path
	}
	if m.Workbench != nil {
		m.Workbench.CancelComparison()
		m.Workbench.BaseRef = ""
		m.Workbench.Staged = staged
		m.Workbench.AgainstHEAD = false
	}
	refresh(m)
}
func paintWorkbench(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w == nil {
		return
	}
	if w.captionReview != nil {
		seat := w.captionReview
		if seat.root == m.WorkspaceRoot && w.Root == seat.root && seat.valid != nil && seat.valid() {
			return // The same review controls are already mounted in the root modal.
		}
		closeCaptionReview(w, seat)
	}
	w.invalidate = c.Services().Invalidate
	if !w.Loaded || w.Root != m.WorkspaceRoot {
		w.cancelBranchList()
		w.cancelRepositoryStatus()
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		w.CancelComparison()
		if w.Loaded && w.Root != m.WorkspaceRoot {
			if m.Text != nil {
				*m.Text = ""
			}
			if m.ErrMsg != nil {
				*m.ErrMsg = ""
			}
			if m.PathFilter != nil {
				*m.PathFilter = ""
			}
		}
		operation := w.Operation
		*w = Workbench{Operation: operation, invalidate: c.Services().Invalidate, AgainstHEAD: true}
		w.reload(m.WorkspaceRoot)
		refresh(m)
	}
	w.finishRepositoryStatus()
	finishComparison(m)
	finishCommitPreparation(m)
	finishGitOperation(m)
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(4, 12, 4, 12).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
			paintDiffScope(c, m)
			if widgets.Button(c, colors, word("比较分支…", "Compare branch…"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(!w.Repository || w.Operation != nil).Clicked() {
				w.CompareOpen = !w.CompareOpen
				if w.CompareOpen {
					branches, err := git.Branches(w.Root)
					w.CompareBranches, w.CompareError = branches, ""
					if err != nil {
						w.CompareError = err.Error()
					}
				}
			}
			if actionButtonWithReason(c, colors, word("提交…", "Commit…"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}, w.commitDisabledReason()) {
				w.CommitOpen = true
			}
			if widgets.Button(c, colors, word("仓库操作", "Repository actions"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
				w.RepositoryToolsOpen = !w.RepositoryToolsOpen
			}
		})
		if w.CompareOpen || w.comparison != nil {
			paintComparison(c, m, fg, muted)
		}
		if w.RepositoryToolsOpen || w.PublishOpen || w.Remote != nil {
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(260).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
					if !paintRepositoryActions(c, m, fg, muted) {
						paintBranches(c, m, fg, muted)
					}
				})
			})
		} else {
			DrawGitProgress(c, w.Progress, fg, muted, func() { w.Progress = nil })
			if !w.Repository {
				ui.Text(c, i18n.T(i18n.KeyDiffNotRepo)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			} else if w.Status != "" {
				ui.Text(c, w.Status).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(12)).SingleLine().Ellipsis("…").TextColor(muted).Tooltip(w.Status)
			}
		}
		paintDiscard(c, m, fg, muted)
		commitDialog(c, m, fg, muted)
	})
}

func paintDiffScope(c *ui.Context, m Model) {
	w := m.Workbench
	caption := word("工作区修改", "Worktree changes")
	if w.BaseRef != "" {
		caption = w.BaseRef + "…HEAD"
	} else if w.Staged {
		caption = word("已暂存", "Staged changes")
	} else if !w.AgainstHEAD {
		caption = word("未暂存", "Unstaged changes")
	}
	ui.MenuButton(c, caption, func(menu *ui.Menu) {
		if menu.Item(word("工作区全部修改", "All worktree changes")).Checked(w.AgainstHEAD && w.BaseRef == "").Chosen() {
			if m.PathFilter != nil {
				*m.PathFilter = ""
			}
			w.CancelComparison()
			w.BaseRef, w.Staged, w.AgainstHEAD = "", false, true
			refresh(m)
		}
		if menu.Item(word("未暂存", "Unstaged changes")).Checked(!w.AgainstHEAD && !w.Staged && w.BaseRef == "").Chosen() {
			scopeDiff(m, "", false)
		}
		if menu.Item(word("已暂存", "Staged changes")).Checked(!w.AgainstHEAD && w.Staged && w.BaseRef == "").Chosen() {
			scopeDiff(m, "", true)
		}
		if len(w.Changes) > 0 {
			menu.Separator()
			for _, change := range w.Changes {
				if menu.Item(change.Path).Checked(m.PathFilter != nil && *m.PathFilter == change.Path).Chosen() {
					scopeDiff(m, change.Path, !change.Unstaged())
				}
			}
		}
	}).Key(m.WorkspaceRoot).Height(28).Disabled(!w.Repository || w.Operation != nil)
}

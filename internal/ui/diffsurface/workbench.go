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
	loader        *workbenchLoader
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
	pendingCaptionAction                      string
	pendingCaptionRoot                        string
	pendingCaptionGen                         uint64
}

func word(zh, en string) string {
	return i18n.Pick(zh, en)
}
func refreshWorkbench(m Model) {
	if m.Workbench != nil {
		w := m.Workbench
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		rootChanged := (w.Root != "" && w.Root != m.WorkspaceRoot) || (w.loader != nil && w.loader.currentRoot() != "" && w.loader.currentRoot() != m.WorkspaceRoot)
		if rootChanged {
			if m.Text != nil {
				*m.Text = ""
			}
			if m.ErrMsg != nil {
				*m.ErrMsg = ""
			}
			if m.PathFilter != nil {
				*m.PathFilter = ""
			}
			w.resetStateForRoot(m.WorkspaceRoot, w.invalidate)
		}
		w.requestReload(m.WorkspaceRoot, true)
	}
	refreshWithForce(m, true)
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
	refreshWithForce(m, true)
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
	w.setInvalidate(c.Services().Invalidate)
	rootChanged := (w.Root != "" && w.Root != m.WorkspaceRoot) || (w.loader != nil && w.loader.currentRoot() != "" && w.loader.currentRoot() != m.WorkspaceRoot)
	if rootChanged {
		if m.Text != nil {
			*m.Text = ""
		}
		if m.ErrMsg != nil {
			*m.ErrMsg = ""
		}
		if m.PathFilter != nil {
			*m.PathFilter = ""
		}
		w.resetStateForRoot(m.WorkspaceRoot, c.Services().Invalidate)
		w.requestReload(m.WorkspaceRoot, true)
		refreshWithForce(m, true)
	} else if !w.Loaded {
		w.bindRoot(m.WorkspaceRoot)
		needDiff := !w.isReloading(m.WorkspaceRoot) && !w.hasReloadResultForRoot(m.WorkspaceRoot)
		w.requestReload(m.WorkspaceRoot, false)
		if needDiff {
			refreshWithForce(m, false)
		}
	} else {
		w.bindRoot(m.WorkspaceRoot)
	}
	w.finishAsync(m)
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Padding(4, 12, 4, 12).Label(word("工作区差异", "Diff panel")).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
			paintDiffScope(c, m)
			if widgets.Button(c, colors, word("比较分支…", "Compare branch…"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(!w.Repository || w.Operation != nil).Clicked() {
				w.CompareOpen = !w.CompareOpen
				if w.CompareOpen {
					w.requestCompareBranches(false)
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
			if !w.Loaded && w.isReloading(m.WorkspaceRoot) {
				ui.Text(c, word("正在加载仓库状态…", "Loading repository status…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			} else if !w.Repository {
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
			refreshWithForce(m, true)
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

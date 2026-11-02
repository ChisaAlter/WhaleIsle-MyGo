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
	}
	w.finishRepositoryStatus()
	finishComparison(m)
	ui.Column(c).Gap(4).Padding(4, 12, 8, 12).Children(func() {
		if paintRepositoryActions(c, m, fg, muted) {
			return
		}
		if !w.CommitOpen && paintCommitPreparation(c, m, fg, muted) {
			return
		}
		paintComparison(c, m, fg, muted)
		paintBranches(c, m, fg, muted)
		ui.Row(c).Gap(6).Children(func() {
			if widgets.Button(c, colors, word("全部修改（对比 HEAD）", "All changes vs HEAD"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
				if m.PathFilter != nil {
					*m.PathFilter = ""
				}
				w.CancelComparison()
				w.BaseRef, w.Staged, w.AgainstHEAD = "", false, true
				refresh(m)
			}
			if widgets.Button(c, colors, word("全部未暂存", "All unstaged"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
				scopeDiff(m, "", false)
			}
			if widgets.Button(c, colors, word("全部已暂存", "All staged"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
				scopeDiff(m, "", true)
			}
		})
		ui.Scroll(c).Height(150).FillWidth().Padding(4).Radius(8).Border(0.5, c.Theme().Border).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				paintCommitChangeRows(c, m, fg, muted)
			})
		})
		paintDiscard(c, m, fg, muted)
		commitLauncher(c, m, fg, muted)
		commitDialog(c, m, fg, muted)
	})
}

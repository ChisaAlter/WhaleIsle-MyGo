package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

// Workbench is window-owned Git UI state. No operation stages or pushes implicitly.
type Workbench struct {
	captionReview                             *captionReviewState
	Progress                                  *git.Progress
	LastStack                                 *git.StackResult
	Remote                                    *remoteWorkflowState
	Commit                                    *commitSelectionState
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
		if paintCommitPreparation(c, m, fg, muted) {
			return
		}
		paintComparison(c, m, fg, muted)
		paintBranches(c, m, fg, muted)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, word("全部修改（对比 HEAD）", "All changes vs HEAD")).Clicked() {
				if m.PathFilter != nil {
					*m.PathFilter = ""
				}
				w.CancelComparison()
				w.BaseRef, w.Staged, w.AgainstHEAD = "", false, true
				refresh(m)
			}
			if ui.Button(c, word("全部未暂存", "All unstaged")).Clicked() {
				scopeDiff(m, "", false)
			}
			if ui.Button(c, word("全部已暂存", "All staged")).Clicked() {
				scopeDiff(m, "", true)
			}
		})
		ui.Text(c, word("勾选本次提交的文件。可单独暂存；编辑器草稿须先保存。", "Check files for this commit. Individual staging is also available. Save editor drafts first.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
		ui.Scroll(c).Height(150).FillWidth().Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for _, change := range w.Changes {
					ch := change
					ui.Row(c).Gap(5).Children(func() {
						paintCommitCheckbox(c, w, ch.Path)
						ui.Text(c, string([]byte{ch.Index, ch.Worktree})).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						if ui.Button(c, ch.Path).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
							scopeDiff(m, ch.Path, !ch.Unstaged())
						}
						if ch.Unstaged() {
							if ui.Button(c, word("丢弃此文件…", "Discard file…")).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
								reviewDiscard(m, ch.Path)
							}
							if ui.Button(c, word("暂存此文件", "Stage file")).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
								paths := []string{ch.Path}
								if ch.Original != "" {
									paths = append(paths, ch.Original)
								}
								_, err := git.StagePaths(m.WorkspaceRoot, paths...)
								w.invalidateCommitReview()
								if err != nil {
									w.Status = err.Error()
								} else {
									w.Status = word("已暂存：", "Staged: ") + ch.Path
								}
								w.reload(m.WorkspaceRoot)
								refresh(m)
							}
						}
						if ch.Staged() {
							if ui.Button(c, word("查看已暂存", "Staged diff")).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
								scopeDiff(m, ch.Path, true)
							}
							if ui.Button(c, word("取消暂存", "Unstage")).FontSize(fonttheme.InterfaceSize(10)).Clicked() {
								paths := []string{ch.Path}
								if ch.Original != "" {
									paths = append(paths, ch.Original)
								}
								_, err := git.UnstagePaths(m.WorkspaceRoot, paths...)
								w.invalidateCommitReview()
								if err != nil {
									w.Status = err.Error()
								} else {
									w.Status = word("已取消暂存：", "Unstaged: ") + ch.Path
								}
								w.reload(m.WorkspaceRoot)
								refresh(m)
							}
						}
					})
				}
			})
		})
		paintDiscard(c, m, fg, muted)
		paintCommitSelection(c, m, fg, muted)
		if w.Status != "" {
			ui.Text(c, w.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
	})
}

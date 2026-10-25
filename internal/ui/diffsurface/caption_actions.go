package diffsurface

import (
	"context"
	"path/filepath"
	"strings"
	"whaleisle/internal/ui/widgets"
	"whaleisle/internal/ui/chrome"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	fonttheme "whaleisle/internal/ui/theme"
)

// RequestCaptionAction opens the existing repository/selected-file review. It
// never prepares a message, runs a model, publishes, pushes or creates a commit.
// Initialize requires CaptionAction's click context. Pull and open_pr are
// handled by the caller's existing GitBadge callbacks.
func (w *Workbench) RequestCaptionAction(root, message, action string) bool {
	if w == nil || w.Operation != nil || strings.TrimSpace(root) == "" {
		return false
	}
	switch action {
	case "commit", "commit_push", "commit_push_pr":
		w.RequestCommit(root, message)
		// The pinned default-ref quick action is named commit_push even on a
		// clean tree. Its commit step is skipped; keep Push review reachable
		// instead of trapping it in an empty selected-file commit form.
		if len(w.Changes) == 0 && action != "commit" {
			if action == "commit_push_pr" {
				w.RequestPR(root)
			} else {
				w.requestRemote(root, "push")
			}
			return true
		}
		s := w.commitState()
		s.Excluded = map[string]bool{}
		s.MakeBranch = false
		s.NewBranch = ""
		s.LargeAcknowledged = false
		s.CaptionAction = action
		s.Push = action != "commit"
		s.WantPR = action == "commit_push_pr"
		return true
	case "push":
		w.requestRemote(root, "push")
		return true
	case "create_pr":
		w.RequestPR(root)
		return true
	case "publish":
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		w.Remote = nil
		if w.Root != root {
			w.cancelBranchList()
			w.cancelRepositoryStatus()
			w.CancelComparison()
			operation, invalidate := w.Operation, w.invalidate
			*w = Workbench{Operation: operation, invalidate: invalidate, AgainstHEAD: true}
		}
		w.reload(root)
		w.PublishOpen = true
		w.PublishReview = false
		if w.PublishOpen {
			w.PublishName = filepath.Base(root)
			w.PublishRemoteURL = ""
			w.PublishPublic = false
		}
		w.Status = ""
		return true
	}
	return false
}

type captionBranchState struct {
	root             string
	open, createOpen bool
	name, notice     string
}

// CaptionBranches attaches the source branch picker to the shell's 27px branch
// trigger. It shares the Workbench's cancellable branch read and operation
// owner; no second Git executor or independent branch cache is introduced.
func CaptionBranches(c *ui.Context, m Model, anchor ui.Element, open *bool) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w == nil || open == nil {
		return
	}
	state := ui.Local(anchor, "caption-branch-owner", func() captionBranchState { return captionBranchState{root: m.WorkspaceRoot} })
	if state.root != m.WorkspaceRoot {
		if state.open {
			w.cancelBranchList()
		}
		*state = captionBranchState{root: m.WorkspaceRoot}
		*open = false
	}
	w.invalidate = c.Services().Invalidate
	finishGitOperation(m)
	if w.Operation != nil {
		*open = false
		state.createOpen = false
	}
	if *open && w.Root != m.WorkspaceRoot {
		w.cancelBranchList()
		w.cancelRepositoryStatus()
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		w.CancelComparison()
		operation, invalidate := w.Operation, w.invalidate
		*w = Workbench{Root: m.WorkspaceRoot, Operation: operation, invalidate: invalidate, AgainstHEAD: true}
	}
	if *open && !state.open {
		w.BranchesOpen = true
		w.BranchQuery = ""
		state.notice = ""
		w.reloadBranches()
	}
	if !*open && state.open {
		w.BranchesOpen = false
		w.BranchQuery = ""
		w.cancelBranchList()
	}
	state.open = *open
	w.finishBranchList()
	if w.BranchStatus != "" && *open && state.notice != w.BranchStatus {
		state.notice = w.BranchStatus
		w.Progress = git.NewProgress(m.WorkspaceRoot, word("分支读取失败", "Failed to load branches"), c.Services().Invalidate)
		w.Progress.Finish(state.notice, true)
	}
	width, height := c.Size()
	ui.PopoverBase(c.Key("caption-branch-picker"), anchor, open, func(panel ui.Element) {
		panel = chrome.PopoverReveal(c, chrome.MenuSurface(panel.Width(max(0, min(float32(320), width-24))).MaxHeight(max(0, height-64)).Padding(6).Gap(2).Role(ui.RoleMenu), c.Theme().Background, c.Theme().Border, 12))
		panel.Children(func() {
			ui.TextInput(c, &w.BranchQuery).FillWidth().Label(word("筛选分支", "Filter branches")).Placeholder(word("搜索分支", "Search branches"))
			if w.branchesRequest != nil {
				ui.Text(c, word("正在读取分支…", "Loading branches…")).Padding(8, 10).FontSize(fonttheme.InterfaceSize(12)).Role(ui.RoleStatus).TextColor(c.Theme().TextMuted)
			}
			if w.BranchStatus != "" {
				ui.Text(c, w.BranchStatus).FillWidth().Padding(8, 10).Role(ui.RoleStatus).TextColor(c.Theme().Danger)
			}
			needle := strings.ToLower(strings.TrimSpace(w.BranchQuery))
			matches := 0
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(0, min(float32(300), height-220))).Children(func() {
				for _, branch := range w.Branches {
					b := branch
					if !strings.Contains(strings.ToLower(b.Name), needle) {
						continue
					}
					matches++
					hint := ""
					if b.Unsupported {
						hint = word("此分支名称含有桌面不支持的字符", "This branch name contains characters unsupported by the desktop")
					}
					row := chrome.MenuRowColors(c, c.Theme().SurfaceHover, c.Theme().SurfacePressed).
						Label(b.Name).Justify(ui.Start).Role(ui.RoleMenuItem).Checked(b.Current).
						Disabled(w.Operation != nil || b.Current || b.Unsupported).Tooltip(hint)
					row.Children(func() {
						ui.Text(c, b.Name).Grow(1).SingleLine().FontSize(13).TextColor(c.Theme().Text)
					})
					if row.Clicked() {
						*open = false
						root, name := m.WorkspaceRoot, b.Name
						beginGitOperation(c, w, word("正在切换分支…", "Switching branch…"), func(ctx context.Context) gitOperationResult {
							out, err := git.SwitchBranchContext(ctx, root, name)
							return gitOperationResult{text: out, err: err}
						})
					}
				}
				if matches == 0 && w.branchesRequest == nil && w.BranchStatus == "" {
					ui.Text(c, word("没有匹配分支", "No matching branches")).FillWidth().Padding(8, 10).FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
				}
			})
			name := strings.TrimSpace(w.BranchQuery)
			taken := captionBranchTaken(w, name)
			label := word("创建新分支…", "Create a new branch…")
			if name != "" {
				label = word("创建分支 ", "Create branch ") + name + "…"
			}
			if ui.Button(c, label).FillWidth().Height(32).Justify(ui.Start).Role(ui.RoleMenuItem).Disabled(w.Operation != nil || name != "" && taken).Clicked() {
				*open = false
				state.name = name
				state.createOpen = true
			}
		})
	})
	if !*open && state.open {
		state.open = false
		w.BranchesOpen = false
		w.BranchQuery = ""
		w.cancelBranchList()
	}
	if state.createOpen {
		ui.DialogBase(c.Key("caption-create-branch"), &state.createOpen, func(backdrop, panel ui.Element) {
			chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(0, min(float32(440), width-32)), 20).Gap(16).
				Label(word("创建分支", "Create branch")).Children(func() {
				ui.Text(c, word("创建分支", "Create branch")).FontSize(fonttheme.InterfaceSize(16)).FontWeight(500)
				ui.Text(c, word("从当前 HEAD 创建并切换到新分支。", "Create a local branch from the current HEAD and switch to it.")).FillWidth().TextColor(c.Theme().TextMuted)
				submitted := ui.TextInput(c, &state.name).FillWidth().Label(word("分支名称", "Branch name")).Placeholder("feature/my-change").Submitted()
				ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
					if widgets.Button(c, colors, word("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
						state.createOpen = false
					}
					name := strings.TrimSpace(state.name)
					canCreate := name != "" && !captionBranchTaken(w, name) && w.Operation == nil
					clicked := ui.PrimaryButton(c, word("创建并切换", "Create and switch")).Disabled(!canCreate).Clicked()
					if canCreate && (clicked || submitted) {
						state.createOpen = false
						state.name = ""
						root := m.WorkspaceRoot
						beginGitOperation(c, w, word("正在创建分支…", "Creating branch…"), func(ctx context.Context) gitOperationResult {
							out, err := git.CreateBranchContext(ctx, root, name)
							return gitOperationResult{text: out, err: err}
						})
					}
				})
			})
		})
	}
}

// CaptionActivity remains mounted even when the ui-git visibility preference
// hides the split button. It settles the same Workbench operation once and
// presents the existing operation-owned progress outside the caption row.
func CaptionActivity(c *ui.Context, m Model) {
	w := m.Workbench
	if w == nil {
		return
	}
	finishGitOperation(m)
	width, height := c.Size()
	if w.Progress != nil {
		ui.Overlay(c.Key("caption-branch-progress"), func() {
			ui.Column(c).Absolute().Bottom(24).Right(24).Width(max(0, min(float32(440), width-48))).MaxHeight(max(0, height-80)).Padding(12).Gap(6).Background(c.Theme().Background).Border(.5, c.Theme().Border).Radius(8).Children(func() {
				DrawGitProgress(c, w.Progress, c.Theme().Text, c.Theme().TextMuted, func() { w.Progress = nil })
			})
		})
	}
}
func captionBranchTaken(w *Workbench, name string) bool {
	for _, branch := range w.Branches {
		if !branch.Remote && branch.Name == name {
			return true
		}
	}
	return false
}

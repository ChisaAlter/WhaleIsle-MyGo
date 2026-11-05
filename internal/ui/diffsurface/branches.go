package diffsurface

import (
	"context"
	"strings"
	"time"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
)

type branchListResult struct {
	branches []git.Branch
	err      error
}

type branchListRequest struct {
	root   string
	cancel context.CancelFunc
	done   chan branchListResult
}

func (w *Workbench) cancelBranchList() {
	if w.branchesRequest != nil {
		w.branchesRequest.cancel()
		w.branchesRequest = nil
	}
}

func (w *Workbench) reloadBranches() {
	w.cancelBranchList()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	request := &branchListRequest{root: w.Root, cancel: cancel, done: make(chan branchListResult, 1)}
	w.branchesRequest = request
	root, invalidate := w.Root, w.invalidate
	w.Branches = nil
	w.BranchStatus = ""
	go func() {
		defer cancel()
		branches, err := git.BranchesContext(ctx, root)
		request.done <- branchListResult{branches: branches, err: err}
		if invalidate != nil {
			invalidate()
		}
	}()
}

func (w *Workbench) finishBranchList() {
	request := w.branchesRequest
	if request == nil {
		return
	}
	if request.root != w.Root || !w.BranchesOpen {
		w.cancelBranchList()
		return
	}
	select {
	case result := <-request.done:
		w.branchesRequest = nil
		if result.err != nil {
			w.BranchStatus = word("分支列表加载失败。", "Failed to load branches: ") + result.err.Error()
		} else {
			w.Branches = result.branches
		}
	default:
	}
}

func paintBranches(c *ui.Context, m Model, fg, muted ui.Color) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w == nil {
		return
	}
	w.finishBranchList()
	ui.Row(c).Gap(6).Children(func() {
		if widgets.Button(c, colors, word("分支…", "Branches…"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
			w.BranchesOpen = !w.BranchesOpen
			if w.BranchesOpen {
				w.reloadBranches()
			} else {
				w.cancelBranchList()
			}
		}
		if w.BranchesOpen && widgets.Button(c, colors, word("刷新分支", "Refresh branches"), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
			w.reloadBranches()
		}
	})
	if !w.BranchesOpen {
		return
	}
	if w.branchesRequest != nil {
		ui.Text(c, word("正在读取分支…", "Loading branches…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	ui.Text(c, word("切换分支会更新磁盘文件；未保存草稿会保留，保存时仍核对磁盘变化。", "Switching updates disk files. Unsaved editor drafts are retained and checked for disk changes on Save.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
	ui.TextInput(c, &w.BranchQuery).FillWidth().Radius(6).Border(0.5, c.Theme().Border).Placeholder(word("筛选分支", "Filter branches"))
	needle := strings.ToLower(strings.TrimSpace(w.BranchQuery))
	ui.Scroll(c).Height(120).FillWidth().Children(func() {
		ui.Column(c).Gap(3).Children(func() {
			matches := 0
			for _, branch := range w.Branches {
				b := branch
				if !strings.Contains(strings.ToLower(b.Name), needle) {
					continue
				}
				matches++
				ui.Row(c).Gap(6).Children(func() {
					label := b.Name
					if b.Current {
						label += word("（当前）", " (current)")
					} else if b.Remote {
						label += word("（远程）", " (remote)")
					}
					ui.Text(c, label).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Grow(1)
					hint := ""
					if b.Unsupported {
						hint = word("此分支名称包含桌面不支持的字符", "This branch name contains characters unsupported by the desktop")
					}
					if widgets.Button(c, colors, word("切换", "Switch"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Disabled(b.Current || b.Unsupported).Tooltip(hint).Clicked() {
						root, name := m.WorkspaceRoot, b.Name
						beginGitOperation(c, w, word("正在切换分支…", "Switching branch…"), func(ctx context.Context) gitOperationResult {
							out, err := git.SwitchBranchContext(ctx, root, name)
							return gitOperationResult{text: out, err: err}
						})
					}
				})
			}
			if matches == 0 && w.BranchStatus == "" && w.branchesRequest == nil {
				ui.Text(c, word("没有匹配的分支", "No matching branches")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	})
	ui.Row(c).Gap(6).Children(func() {
		ui.TextInput(c, &w.BranchName).Grow(1).Radius(6).Border(0.5, c.Theme().Border).Placeholder(word("新分支名称", "New branch name"))
		taken := false
		for _, branch := range w.Branches {
			if !branch.Remote && branch.Name == strings.TrimSpace(w.BranchName) {
				taken = true
				break
			}
		}
		if widgets.Button(c, colors, word("创建并切换", "Create and switch"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall}).Disabled(strings.TrimSpace(w.BranchName) == "" || taken).Clicked() {
			root, name := m.WorkspaceRoot, strings.TrimSpace(w.BranchName)
			beginGitOperation(c, w, word("正在创建分支…", "Creating branch…"), func(ctx context.Context) gitOperationResult {
				out, err := git.CreateBranchContext(ctx, root, name)
				return gitOperationResult{text: out, err: err}
			})
		}
	})
	if w.BranchStatus != "" {
		ui.Text(c, w.BranchStatus).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
}

package diffsurface

import (
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
)

func (w *Workbench) reloadBranches() {
	branches, err := git.Branches(w.Root)
	if err != nil {
		w.Branches = nil
		w.BranchStatus = word("分支列表读取失败：", "Failed to load branches: ") + err.Error()
		return
	}
	w.Branches = branches
	w.BranchStatus = ""
}

func branchChanged(m Model, out string, err error) {
	w := m.Workbench
	w.ReviewToken = ""
	w.ReviewText = ""
	if err != nil {
		w.BranchStatus = err.Error()
		return
	}
	w.BranchName = ""
	w.BranchQuery = ""
	w.reload(m.WorkspaceRoot)
	w.reloadBranches()
	w.Status = out
	scopeDiff(m, "", false)
}

func paintBranches(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil {
		return
	}
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, word("分支…", "Branches…")).Clicked() {
			w.BranchesOpen = !w.BranchesOpen
			if w.BranchesOpen {
				w.reloadBranches()
			}
		}
		if w.BranchesOpen && ui.Button(c, word("刷新分支", "Refresh branches")).Clicked() {
			w.reloadBranches()
		}
	})
	if !w.BranchesOpen {
		return
	}
	ui.Text(c, word("切换分支会更新磁盘文件；未保存草稿会保留，保存时仍核对磁盘变化。", "Switching updates disk files. Unsaved editor drafts are retained and checked for disk changes on Save.")).FontSize(fonttheme.InterfaceSize(10)).TextColor(muted)
	ui.TextInput(c, &w.BranchQuery).FillWidth().Placeholder(word("筛选分支", "Filter branches"))
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
					if ui.Button(c, word("切换", "Switch")).FontSize(fonttheme.InterfaceSize(11)).Disabled(b.Current).Clicked() {
						out, err := git.SwitchBranch(m.WorkspaceRoot, b.Name)
						branchChanged(m, out, err)
					}
				})
			}
			if matches == 0 && w.BranchStatus == "" {
				ui.Text(c, word("没有匹配分支", "No matching branches")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	})
	ui.Row(c).Gap(6).Children(func() {
		ui.TextInput(c, &w.BranchName).Grow(1).Placeholder(word("新分支名称", "New branch name"))
		if ui.Button(c, word("创建并切换", "Create and switch")).Disabled(strings.TrimSpace(w.BranchName) == "").Clicked() {
			out, err := git.CreateBranch(m.WorkspaceRoot, strings.TrimSpace(w.BranchName))
			branchChanged(m, out, err)
		}
	})
	if w.BranchStatus != "" {
		ui.Text(c, w.BranchStatus).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	}
}

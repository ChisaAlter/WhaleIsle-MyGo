package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	fonttheme "whaleisle/internal/ui/theme"
)

func reviewDiscard(m Model, path string) {
	w := m.Workbench
	plan, err := git.PrepareDiscard(m.WorkspaceRoot, path)
	if err != nil {
		w.Status = err.Error()
		w.Discard = nil
		return
	}
	w.Discard = &plan
	w.Status = ""
}

func paintDiscard(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w.Discard == nil {
		return
	}
	plan := *w.Discard
	ui.Column(c).Gap(5).Padding(8, 8, 8, 8).Children(func() {
		ui.Text(c, word("确认丢弃：", "Confirm discard: ")+plan.Path).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
		impact := word("此文件的未暂存磁盘修改将被已暂存版本覆盖，无法撤销。已暂存内容及其他文件保留。未保存的编辑器草稿不会被删除。", "Unstaged disk edits to this file will be overwritten by its staged version and cannot be undone. Staged changes and other files are preserved. Unsaved editor drafts are retained.")
		if plan.Untracked {
			impact = word("将永久删除此未跟踪文件，无法通过 Git 恢复。其他文件保留。未保存的编辑器草稿不会被删除。", "This untracked file will be permanently deleted and cannot be recovered through Git. Other files and unsaved editor drafts are retained.")
		}
		ui.Text(c, impact).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, word("取消", "Cancel")).Clicked() {
				w.Discard = nil
			}
			if ui.Button(c, word("确认丢弃此文件", "Confirm discard file")).Clicked() {
				// Capture and clear before running: repeated clicks cannot replay this review.
				w.Discard = nil
				_, err := git.DiscardReviewed(m.WorkspaceRoot, plan)
				if err != nil {
					w.Status = err.Error()
					return
				}
				w.ReviewToken = ""
				w.ReviewText = ""
				w.reload(m.WorkspaceRoot)
				refresh(m)
				w.Status = word("已丢弃：", "Discarded: ") + plan.Path
			}
		})
	})
}

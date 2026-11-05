package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w.Discard == nil {
		return
	}
	plan := *w.Discard
	open := true
	ui.DialogBase(c.Key("discard-file"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 480, 20).MaxHeightPercent(88).Gap(12).Children(func() {
			ui.Text(c, word("还原工作区更改？", "Discard workspace changes?")).FontSize(fonttheme.InterfaceSize(16)).Bold().TextColor(fg)
			impact := word("此文件的未暂存磁盘修改将被已暂存版本覆盖，无法撤销。已暂存内容及其他文件保留。未保存的编辑器草稿不会被删除。", "Unstaged disk edits to this file will be overwritten by its staged version and cannot be undone. Staged changes and other files are preserved. Unsaved editor drafts are retained.")
			if plan.Untracked {
				impact = word("将永久删除此未跟踪文件，无法通过 Git 恢复。其他文件保留。未保存的编辑器草稿不会被删除。", "This untracked file will be permanently deleted and cannot be recovered through Git. Other files and unsaved editor drafts are retained.")
			}
			ui.Scroll(c).FillWidth().MinHeight(0).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					ui.Text(c, plan.Path).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
					ui.Text(c, impact).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				})
			})
			ui.Row(c).FillWidth().Wrap().Justify(ui.End).Gap(6).Children(func() {
				if widgets.Button(c, colors, word("取消", "Cancel"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					w.Discard = nil
				}
				if widgets.Button(c, colors, word("还原", "Discard"), widgets.ButtonSpec{Variant: widgets.ButtonDanger, Size: widgets.ButtonSmall}).Clicked() {
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
					w.Status = word("已还原：", "Discarded: ") + plan.Path
				}
			})
		})
	})
	if !open {
		w.Discard = nil
	}
}

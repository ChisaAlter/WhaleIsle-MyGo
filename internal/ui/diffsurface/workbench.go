package diffsurface

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

// Workbench is window-owned Git UI state. No operation stages or pushes implicitly.
type Workbench struct {
	Root                                     string
	Changes                                  []git.Change
	Staged                                   bool
	Message, Status, ReviewToken, ReviewText string
	Loaded                                   bool
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
	changes, err := git.Changes(root)
	if err != nil {
		w.Status = err.Error()
		w.Changes = nil
		return
	}
	w.Changes = changes
}
func refreshWorkbench(m Model) {
	if m.Workbench != nil {
		m.Workbench.reload(m.WorkspaceRoot)
	}
	refresh(m)
}
func scopeDiff(m Model, path string, staged bool) {
	if m.PathFilter != nil {
		*m.PathFilter = path
	}
	if m.Workbench != nil {
		m.Workbench.Staged = staged
	}
	refresh(m)
}
func paintWorkbench(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	if w == nil {
		return
	}
	if !w.Loaded || w.Root != m.WorkspaceRoot {
		*w = Workbench{}
		w.reload(m.WorkspaceRoot)
	}
	ui.Column(c).Gap(4).Padding(4, 12, 8, 12).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c, word("全部未暂存", "All unstaged")).Clicked() {
				scopeDiff(m, "", false)
			}
			if ui.Button(c, word("全部已暂存", "All staged")).Clicked() {
				scopeDiff(m, "", true)
			}
		})
		ui.Text(c, word("单文件暂存；提交仅包含已暂存内容。编辑器草稿须先保存。", "Stage individual files; commits contain only staged changes. Save editor drafts first.")).FontSize(10).TextColor(muted)
		ui.Scroll(c).Height(150).FillWidth().Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for _, change := range w.Changes {
					ch := change
					ui.Row(c).Gap(5).Children(func() {
						ui.Text(c, string([]byte{ch.Index, ch.Worktree})).FontSize(11).TextColor(muted)
						if ui.Button(c, ch.Path).FontSize(11).Clicked() {
							scopeDiff(m, ch.Path, !ch.Unstaged())
						}
						if ch.Unstaged() {
							if ui.Button(c, word("暂存此文件", "Stage file")).FontSize(10).Clicked() {
								paths := []string{ch.Path}
								if ch.Original != "" {
									paths = append(paths, ch.Original)
								}
								_, err := git.StagePaths(m.WorkspaceRoot, paths...)
								w.ReviewToken = ""
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
							if ui.Button(c, word("查看已暂存", "Staged diff")).FontSize(10).Clicked() {
								scopeDiff(m, ch.Path, true)
							}
							if ui.Button(c, word("取消暂存", "Unstage")).FontSize(10).Clicked() {
								paths := []string{ch.Path}
								if ch.Original != "" {
									paths = append(paths, ch.Original)
								}
								_, err := git.UnstagePaths(m.WorkspaceRoot, paths...)
								w.ReviewToken = ""
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
		ui.TextInput(c, &w.Message).FillWidth().Placeholder(word("提交说明（仅已暂存）", "Commit message (staged only)"))
		if w.ReviewToken == "" {
			if ui.Button(c, word("核对已暂存内容并提交…", "Review staged changes to commit…")).Clicked() {
				if strings.TrimSpace(w.Message) == "" {
					w.Status = word("请填写提交说明", "Enter a commit message")
				} else {
					text, token, err := git.IndexReview(m.WorkspaceRoot)
					if err != nil {
						w.Status = err.Error()
					} else {
						w.ReviewToken = token
						w.ReviewText = text
						scopeDiff(m, "", true)
						w.Status = word("核对下方完整已暂存diff，确认后创建本地提交，不会推送。", "Review the complete staged diff below. Confirm creates a local commit; it does not push.")
					}
				}
			}
		} else {
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, word("确认提交已暂存内容", "Confirm staged commit")).Clicked() {
					out, err := git.CommitReviewed(m.WorkspaceRoot, w.Message, w.ReviewToken)
					w.ReviewToken = ""
					if err != nil {
						w.Status = err.Error()
					} else {
						w.Status = out
						w.Message = ""
					}
					w.reload(m.WorkspaceRoot)
					refresh(m)
				}
				if ui.Button(c, word("取消", "Cancel")).Clicked() {
					w.ReviewToken = ""
					w.Status = ""
				}
			})
		}
		if w.Status != "" {
			ui.Text(c, w.Status).FontSize(11).TextColor(fg)
		}
	})
}

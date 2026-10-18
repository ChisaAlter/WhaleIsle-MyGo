package diffsurface

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

type comparisonResult struct {
	text string
	err  error
}
type comparisonRequest struct {
	root, path, baseRef string
	cancel              context.CancelFunc
	done                chan comparisonResult
}

// CancelComparison also invalidates the request identity, so a late result can
// never replace a newer worktree scope or another repository's diff.
func (w *Workbench) CancelComparison() {
	if w.comparison != nil {
		w.comparison.cancel()
		w.comparison = nil
	}
}

func beginComparison(m Model, path string) {
	w := m.Workbench
	w.CancelComparison()
	ctx, cancel := context.WithCancel(context.Background())
	request := &comparisonRequest{root: m.WorkspaceRoot, path: path, baseRef: w.BaseRef, cancel: cancel, done: make(chan comparisonResult, 1)}
	w.comparison = request
	if m.Text != nil {
		*m.Text = ""
	}
	if m.ErrMsg != nil {
		*m.ErrMsg = ""
	}
	invalidate := w.invalidate
	go func() {
		text, err := git.CompareBranch(ctx, request.root, request.path, request.baseRef)
		request.done <- comparisonResult{text: text, err: err}
		if invalidate != nil {
			invalidate()
		}
	}()
}

func finishComparison(m Model) {
	w := m.Workbench
	request := w.comparison
	if request == nil {
		return
	}
	path := ""
	if m.PathFilter != nil {
		path = *m.PathFilter
	}
	if request.root != m.WorkspaceRoot || request.root != w.Root || request.baseRef != w.BaseRef || request.path != path {
		w.CancelComparison()
		return
	}
	select {
	case result := <-request.done:
		w.CancelComparison()
		if m.Text != nil {
			*m.Text = result.text
		}
		errMsg := ""
		if result.err != nil {
			errMsg = result.err.Error()
		} else if result.text == "" {
			errMsg = i18n.T(i18n.KeyDiffEmpty)
		}
		if m.ErrMsg != nil {
			*m.ErrMsg = errMsg
		}
	default:
	}
}

func paintComparison(c *ui.Context, m Model, fg, muted ui.Color) {
	w := m.Workbench
	label := word("比较基线…", "Compare branch…")
	if w.BaseRef != "" {
		label = word("基线：", "Base: ") + w.BaseRef
	}
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, label).Clicked() {
			w.CompareOpen = !w.CompareOpen
			if w.CompareOpen {
				branches, err := git.Branches(w.Root)
				w.CompareBranches = branches
				w.CompareError = ""
				if err != nil {
					w.CompareError = err.Error()
				}
			}
		}
		if w.BaseRef != "" && ui.Button(c, word("返回工作区差异", "Worktree changes")).Clicked() {
			scopeDiff(m, "", false)
		}
		if w.comparison != nil {
			ui.Text(c, word("正在比较…", "Comparing…")).FontSize(11).TextColor(muted)
			if ui.Button(c, word("取消比较", "Cancel comparison")).Clicked() {
				w.CancelComparison()
				if m.ErrMsg != nil {
					*m.ErrMsg = word("已取消比较；可刷新重试。", "Comparison canceled; refresh to retry.")
				}
			}
		}
	})
	if w.BaseRef != "" {
		ui.Text(c, w.BaseRef+"...HEAD · "+word("仅比较共同祖先以来的提交，不含未提交修改。", "Committed changes since the common ancestor; local edits are excluded.")).FontSize(10).TextColor(muted)
	}
	if !w.CompareOpen {
		return
	}
	ui.TextInput(c, &w.CompareQuery).FillWidth().Placeholder(word("筛选比较分支", "Filter comparison branches"))
	needle := strings.ToLower(strings.TrimSpace(w.CompareQuery))
	ui.Scroll(c).Height(110).FillWidth().Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			for _, branch := range w.CompareBranches {
				b := branch
				if !strings.Contains(strings.ToLower(b.Name), needle) {
					continue
				}
				if ui.Button(c, b.Name).FontSize(11).Clicked() {
					w.BaseRef = b.Name
					w.Staged = false
					w.CompareOpen = false
					if m.PathFilter != nil {
						*m.PathFilter = ""
					}
					refresh(m)
				}
			}
		})
	})
	if w.CompareError != "" {
		ui.Text(c, w.CompareError).FontSize(11).TextColor(fg)
	}
}

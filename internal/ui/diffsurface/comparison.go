package diffsurface

import (
	"context"
	"strings"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

type comparisonResult struct {
	text     string
	err      error
	prepared *preparedPatchDoc
}
type comparisonRequest struct {
	root, path, baseRef string
	gen                 uint64
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
	beginComparisonForce(m, path, false)
}

func beginComparisonForce(m Model, path string, force bool) {
	w := m.Workbench
	if w == nil {
		return
	}
	gen := w.bindRoot(m.WorkspaceRoot)
	if !force && w.comparison != nil &&
		w.comparison.root == m.WorkspaceRoot &&
		w.comparison.path == path &&
		w.comparison.baseRef == w.BaseRef &&
		(w.comparison.gen == 0 || w.comparison.gen == gen) {
		return
	}
	w.CancelComparison()
	ctx, cancel := context.WithCancel(context.Background())
	request := &comparisonRequest{
		root:    m.WorkspaceRoot,
		path:    path,
		baseRef: w.BaseRef,
		gen:     gen,
		cancel:  cancel,
		done:    make(chan comparisonResult, 1),
	}
	w.comparison = request
	if m.Text != nil {
		*m.Text = ""
	}
	if m.ErrMsg != nil {
		*m.ErrMsg = ""
	}
	loader := w.loader
	invalidate := w.invalidate
	viewSplit, viewWrap := w.View.Split, w.View.Wrap
	ops := gitOps
	go func() {
		defer cancel()
		text, err := ops.compareBranch(ctx, request.root, request.path, request.baseRef)
		var prepared *preparedPatchDoc
		if err == nil && strings.TrimSpace(text) != "" {
			prepared = prebuildPatchDocument(text, viewSplit, viewWrap)
		}
		request.done <- comparisonResult{text: text, err: err, prepared: prepared}
		if loader != nil {
			loader.notify()
		} else if invalidate != nil {
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
	if request.root != m.WorkspaceRoot || request.root != w.Root || request.baseRef != w.BaseRef || request.path != path || (request.gen != 0 && request.gen != w.currentGen()) {
		w.CancelComparison()
		return
	}
	select {
	case result := <-request.done:
		w.CancelComparison()
		if m.Text != nil {
			*m.Text = result.text
		}
		if result.prepared != nil {
			w.View.applyPrepared(result.prepared)
		}
		errMsg := ""
		if result.err != nil {
			errMsg = errcopy.Sentence(result.err)
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	label := word("比较基线…", "Compare branch…")
	if w.BaseRef != "" {
		label = word("基线：", "Base: ") + w.BaseRef
	}
	ui.Row(c).Gap(6).Children(func() {
		if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
			w.CompareOpen = !w.CompareOpen
			if w.CompareOpen {
				w.requestCompareBranches(false)
			}
		}
		if w.BaseRef != "" && widgets.Button(c, colors, word("返回工作区差异", "Worktree changes"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).Clicked() {
			scopeDiff(m, "", false)
		}
		if w.comparison != nil {
			ui.Text(c, word("正在比较…", "Comparing…")).FontSize(11).TextColor(muted)
			if widgets.Button(c, colors, word("取消比较", "Cancel comparison"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
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
	coss.Input(c, &w.CompareQuery, coss.InputOptions{Size: coss.InputSM, Placeholder: word("筛选比较分支", "Filter comparison branches")})
	needle := strings.ToLower(strings.TrimSpace(w.CompareQuery))
	ui.Scroll(c).Height(110).FillWidth().Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			if len(w.CompareBranches) == 0 && w.isCompareBranchesLoading() {
				ui.Text(c, word("正在加载分支列表…", "Loading branches…")).FontSize(11).TextColor(muted)
			} else {
				for _, branch := range w.CompareBranches {
					b := branch
					if !strings.Contains(strings.ToLower(b.Name), needle) {
						continue
					}
					if coss.Button(c, b.Name, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
						w.BaseRef = b.Name
						w.Staged = false
						w.CompareOpen = false
						if m.PathFilter != nil {
							*m.PathFilter = ""
						}
						refreshWithForce(m, true)
					}
				}
			}
		})
	})
	if w.CompareError != "" {
		ui.Text(c, w.CompareError).FontSize(11).TextColor(fg)
	}
}

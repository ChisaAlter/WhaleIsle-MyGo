package markdown

import (
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// richBlock retains span positions so native controls replace the corresponding
// text in place, without reordering surrounding prose or inline file targets.
type richBlock struct {
	start, end int
	node       ast.Node
}

func viewBlocks(c *ui.Context, source string, st Style, spans []ui.Span, targets map[int]string, blocks []richBlock) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		at := 0
		for _, block := range blocks {
			start, end := min(block.start, len(spans)), min(block.end, len(spans))
			viewSpans(c, st, spans[at:start], targets, at)
			switch node := block.node.(type) {
			case *extast.Table:
				viewTable(c, node, source, st)
			default:
				viewCodeBlock(c, node, source, st)
			}
			at = end
		}
		viewSpans(c, st, spans[at:], targets, at)
	})
}

func codeBlockSource(node ast.Node, source string) (language, code string) {
	if fenced, ok := node.(*ast.FencedCodeBlock); ok {
		language = string(fenced.Language([]byte(source)))
	}
	var buf strings.Builder
	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		buf.Write(line.Value([]byte(source)))
	}
	return language, buf.String()
}

func viewCodeBlock(c *ui.Context, node ast.Node, source string, st Style) {
	language, code := codeBlockSource(node, source)
	if st.RenderFence != nil && node.Lines().Len() > 0 && st.RenderFence(c, language, code, node.Lines().At(0).Start) {
		return
	}
	root := ui.Column(c).FillWidth().MinWidth(0).Background(st.CodeBG).Padding(8).Gap(4)
	wrap := ui.Local(root, "wrap-code", func() bool { return st.CodeWrapDefault })
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			label := language
			if label == "" {
				label = "code"
			}
			ui.Text(c, label).FontSize(st.FontSize).TextColor(st.Muted).Grow(1)
			if ui.Button(c, "Copy code").Clicked() {
				c.WriteClipboard(code)
			}
			label = "Wrap lines"
			if *wrap {
				label = "Unwrap lines"
			}
			if ui.Button(c, label).Clicked() {
				*wrap = !*wrap
			}
		})
		spans := highlightedCodeSpans(language, code, st)
		if *wrap {
			ui.RichText(c, spans...).FillWidth().MinWidth(0).Font(theme.CurrentPreferences().Code.Family).FontSize(theme.CurrentPreferences().Code.Size).TextColor(st.FG)
		} else {
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				ui.RichText(c, spans...).NoWrap().Font(theme.CurrentPreferences().Code.Family).FontSize(theme.CurrentPreferences().Code.Size).TextColor(st.FG)
			})
		}
	})
}

func viewTable(c *ui.Context, table *extast.Table, source string, st Style) {
	// Fixed column widths keep cells aligned across rows; horizontal scrolling
	// retains wide tables without squeezing or truncating their content.
	ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).Children(func() {
			for row := table.FirstChild(); row != nil; row = row.NextSibling() {
				_, header := row.(*extast.TableHeader)
				ui.Row(c).Children(func() {
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						ui.Column(c).Width(180).Padding(6).BorderWidth(1).BorderColor(st.Muted).Children(func() {
							cellStyle := st
							cellStyle.maths = map[int]mathFormula{}
							targets := map[int]string{}
							cellStyle.codeSpanAt = func(index int, value string) {
								if st.ResolveFile != nil && st.OnOpenFile != nil {
									targets[index] = st.ResolveFile(value)
								}
							}
							var spans []ui.Span
							walkInlines(cell, source, cellStyle, &spans)
							if header {
								for i := range spans {
									spans[i].Weight = 700
								}
							}
							viewSpans(c, cellStyle, spans, targets, 0)
						})
					}
				})
			}
		})
	})
}

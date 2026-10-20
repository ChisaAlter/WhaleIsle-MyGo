package markdown

import (
	"strconv"
	"strings"

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
		for blockIndex, block := range blocks {
			start, end := min(block.start, len(spans)), min(block.end, len(spans))
			viewSpans(c, st, spans[at:start], targets, at)
			switch node := block.node.(type) {
			case *extast.Table:
				viewTable(c, node, source, st)
			case *ast.Image:
				// The URL may repeat, and an empty alt has no span position. Each
				// source occurrence owns a parent scope independent of URL/alt.
				ui.Column(c.Key("markdown-image-" + strconv.Itoa(blockIndex))).FillWidth().MinWidth(0).Children(func() {
					var alt strings.Builder
					for _, span := range spans[start:end] {
						alt.WriteString(span.Text)
					}
					render := func(style Style, files map[int]string) {
						if st.RenderImage == nil || !st.RenderImage(c, string(node.Destination), alt.String()) {
							fallback := spans[start:end]
							if st.DocumentLayout {
								fallback = []ui.Span{{Text: alt.String(), Size: st.FontSize, Color: st.Muted, Italic: true}}
							}
							viewSpans(c, style, fallback, files, start)
						}
					}
					if href := enclosingImageURL(node); href != "" && st.OnOpenURL != nil {
						label := alt.String()
						if label == "" {
							label = href
						}
						link := ui.Box(c).FillWidth().MinWidth(0).Role(ui.RoleLink).Focusable().Label(label).Tooltip(href).Cursor(ui.CursorPointer)
						link.Children(func() {
							fallback := st
							fallback.urlTargets = nil // The enclosing image link owns this activation.
							render(fallback, nil)
						})
						if link.Clicked() || link.Shortcut(0, ui.KeyEnter) || link.Shortcut(0, ui.KeySpace) {
							st.OnOpenURL(href)
						}
					} else {
						render(st, targets)
					}
				})
			default:
				viewCodeBlock(c, node, source, st)
			}
			at = end
		}
		viewSpans(c, st, spans[at:], targets, at)
	})
}

func enclosingImageURL(image *ast.Image) string {
	for parent := image.Parent(); parent != nil; parent = parent.Parent() {
		if link, ok := parent.(*ast.Link); ok {
			return documentURL(string(link.Destination))
		}
		if parent.Type() == ast.TypeBlock {
			break
		}
	}
	return ""
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

// viewCodeBlock and viewTable live in codeblock.go and table.go: both draw
// block-level chrome that a single RichText run cannot express.

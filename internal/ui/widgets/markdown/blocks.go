package markdown

import (
	"strconv"
	"strings"

	"whaleisle/internal/markdownsyntax"

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
			case *markdownsyntax.ImagePlaceholder:
				if st.RenderImage == nil || !st.RenderImage(c, node.Destination, "") {
					viewSpans(c, st, spans[start:end], targets, start)
				}
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
					if destination := enclosingImageLink(node); destination != "" {
						href := documentURL(destination)
						path, line, file := ParseFileLink(destination)
						if line > 0 {
							path += "#L" + strconv.Itoa(line)
						}
						active := href != "" && st.OnOpenURL != nil || file && st.OnOpenFile != nil && !st.Streaming
						if !active {
							render(st, targets)
							return
						}
						label := alt.String()
						if label == "" {
							label = destination
						}
						link := ui.Box(c).FillWidth().MinWidth(0).Role(ui.RoleLink).Focusable().Label(label).Tooltip(destination).Cursor(ui.CursorPointer)
						link.Children(func() {
							fallback := st
							fallback.urlTargets = nil // The enclosing image link owns this activation.
							render(fallback, nil)
						})
						if externalURLActivated(link) {
							if href != "" {
								st.OnOpenURL(href)
							} else {
								st.OnOpenFile(path)
							}
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

func enclosingImageLink(image *ast.Image) string {
	for parent := image.Parent(); parent != nil; parent = parent.Parent() {
		if link, ok := parent.(*ast.Link); ok {
			return string(link.Destination)
		}
		if parent.Type() == ast.TypeBlock {
			break
		}
	}
	return ""
}

func codeBlockSource(node ast.Node, source string) (language, code string) {
	if fenced, ok := node.(*ast.FencedCodeBlock); ok {
		language = fencedLanguage(fenced, source)
	}
	lines := node.Lines()
	n := lines.Len()
	if n == 0 {
		return language, ""
	}
	if n == 1 {
		return language, segmentText(lines.At(0), source)
	}
	first := lines.At(0)
	last := lines.At(n - 1)
	contiguous := first.Padding == 0 && !first.ForceNewline && first.Stop <= len(source)
	prevStop := first.Stop
	totalLen := first.Stop - first.Start + first.Padding
	for i := 1; i < n; i++ {
		seg := lines.At(i)
		if seg.Start != prevStop || seg.Padding != 0 || seg.ForceNewline || seg.Stop > len(source) {
			contiguous = false
		}
		prevStop = seg.Stop
		totalLen += seg.Stop - seg.Start + seg.Padding
	}
	if contiguous && first.Start >= 0 && last.Stop <= len(source) && first.Start <= last.Stop {
		return language, source[first.Start:last.Stop]
	}
	var buf strings.Builder
	buf.Grow(max(0, totalLen))
	for i := 0; i < n; i++ {
		buf.WriteString(segmentText(lines.At(i), source))
	}
	return language, buf.String()
}

// viewCodeBlock and viewTable live in codeblock.go and table.go: both draw
// block-level chrome that a single RichText run cannot express.

package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type mathFormula struct {
	TeX, Raw string
	Display  bool
}

var inlineMathKind = ast.NewNodeKind("InlineMath")
var blockMathKind = ast.NewNodeKind("BlockMath")

type inlineMath struct {
	ast.BaseInline
	formula mathFormula
}

func (n *inlineMath) Kind() ast.NodeKind            { return inlineMathKind }
func (n *inlineMath) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

type blockMath struct {
	ast.BaseBlock
	formula mathFormula
	close   string
	closed  bool
}

func (n *blockMath) Kind() ast.NodeKind            { return blockMathKind }
func (n *blockMath) IsRaw() bool                   { return true }
func (n *blockMath) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

func newMarkdown() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(
		parser.WithInlineParsers(util.Prioritized(mathInlineParser{}, 50)),
		parser.WithBlockParsers(util.Prioritized(mathBlockParser{}, 50)),
	))
}

type mathInlineParser struct{}

func (mathInlineParser) Trigger() []byte { return []byte{'$', '\\'} }
func (mathInlineParser) Parse(_ ast.Node, reader text.Reader, _ parser.Context) ast.Node {
	line, _ := reader.PeekLine()
	open, close, display := "", "", false
	switch {
	case strings.HasPrefix(string(line), "$$"):
		open, close, display = "$$", "$$", true
	case strings.HasPrefix(string(line), "$"):
		open, close = "$", "$"
	case strings.HasPrefix(string(line), `\(`):
		open, close = `\(`, `\)`
	case strings.HasPrefix(string(line), `\[`):
		open, close, display = `\[`, `\]`, true
	default:
		return nil
	}
	startLine, start := reader.Position()
	reader.Advance(len(open))
	var body strings.Builder
	for {
		line, _ := reader.PeekLine()
		if line == nil {
			break
		}
		for i := 0; i < len(line); i++ {
			if strings.HasPrefix(string(line[i:]), close) {
				// Currency-like spaces are not inline dollar delimiters.
				candidate := body.String() + string(line[:i])
				if open == "$" && (candidate == "" || strings.TrimSpace(candidate) != candidate) {
					continue
				}
				reader.Advance(i + len(close))
				return &inlineMath{formula: mathFormula{TeX: candidate, Raw: open + candidate + close, Display: display}}
			}
			if line[i] == '\\' && i+1 < len(line) {
				i++
			}
		}
		body.Write(line)
		reader.AdvanceLine()
	}
	reader.SetPosition(startLine, start)
	return nil
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$', '\\'} }
func (mathBlockParser) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || pos >= len(line) {
		return nil, parser.NoChildren
	}
	value := strings.TrimSpace(string(line[pos:]))
	open, close := "", ""
	if strings.HasPrefix(value, "$$") {
		open, close = "$$", "$$"
	} else if strings.HasPrefix(value, `\[`) {
		open, close = `\[`, `\]`
	} else {
		return nil, parser.NoChildren
	}
	rest := value[len(open):]
	n := &blockMath{close: close, formula: mathFormula{Raw: string(line[pos:]), Display: true}}
	if end := strings.Index(rest, close); end >= 0 {
		if strings.TrimSpace(rest[end+len(close):]) != "" {
			return nil, parser.NoChildren
		}
		n.formula.TeX, n.closed = rest[:end], true
	} else {
		n.formula.TeX = rest + "\n"
	}
	return n, parser.NoChildren
}
func (mathBlockParser) Continue(node ast.Node, reader text.Reader, _ parser.Context) parser.State {
	n := node.(*blockMath)
	if n.closed {
		return parser.Close
	}
	line, segment := reader.PeekLine()
	value := string(line)
	if end := strings.Index(value, n.close); end >= 0 && strings.TrimSpace(value[end+len(n.close):]) == "" {
		n.formula.TeX += value[:end]
		n.formula.Raw += value
		n.closed = true
		newline := 0
		if strings.HasSuffix(value, "\n") {
			newline = 1
		}
		reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
		return parser.Close
	}
	n.formula.TeX += value
	n.formula.Raw += value
	return parser.Continue | parser.NoChildren
}
func (mathBlockParser) Close(ast.Node, text.Reader, parser.Context) {}
func (mathBlockParser) CanInterruptParagraph() bool                 { return true }
func (mathBlockParser) CanAcceptIndentedLine() bool                 { return false }

package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func parsedBlockText(blocks []documentBlock, streaming bool) string {
	var spans []ui.Span
	for _, block := range blocks {
		holder := ast.NewDocument()
		holder.AppendChild(holder, block.node)
		walkBlocks(holder, block.source, Style{Streaming: streaming}, &spans)
	}
	trimTrailingNewline(&spans)
	return concat(spans)
}

func fullParsedText(source string, streaming bool) string {
	grammar := markdownGrammar(streaming)
	doc := grammar.Parser().Parse(text.NewReader([]byte(source)))
	var spans []ui.Span
	walkBlocks(doc, source, Style{Streaming: streaming}, &spans)
	trimTrailingNewline(&spans)
	return concat(spans)
}

func TestStreamingAppendReshapesOnlyUnstableBlocksAndSettlesReferences(t *testing.T) {
	var state streamDocument
	source := "# Fixed\n\nIntro\n\nName | Value"
	state.update(source, true)
	source += "\n--- | ---\nfirst | 1\n\n- item\n"
	blocks := state.update(source, true)
	table := false
	for _, block := range blocks {
		if _, ok := block.node.(*extast.Table); ok {
			table = true
		}
	}
	if !table {
		t.Fatal("streaming paragraph did not become a table")
	}
	source += "\n- next\n\nLast [source][ref].\n\n[ref]: https://example.invalid\n"
	blocks = state.update(source, true)
	keys := map[int]bool{}
	for _, block := range blocks {
		if keys[block.key] {
			t.Fatalf("repeated source key %d", block.key)
		}
		keys[block.key] = true
	}
	settled := state.update(source, false)
	if got, want := parsedBlockText(settled, false), concat(Spans(source, Style{})); got != want {
		t.Fatalf("settled parse=%q, full=%q", got, want)
	}
	state.update("Replacement", true)
	if len(state.frozen) != 0 {
		t.Fatal("edited text retained previous prefix")
	}
}

func TestStreamingLongReplyParsesBoundedTail(t *testing.T) {
	var state streamDocument
	var source string
	for index := range 200 {
		source += fmt.Sprintf("Paragraph %03d stays in the transcript.\n\n", index)
		state.update(source, true)
	}
	if state.parsedBytes > len(source)*8 {
		t.Fatalf("stream reparsed the complete reply: parsed=%d source=%d", state.parsedBytes, len(source))
	}
	t.Logf("200 appended paragraphs: source=%d grammar-input=%d", len(source), state.parsedBytes)
	before := state.parsedBytes
	state.update(source, true)
	if state.parsedBytes != before {
		t.Fatal("identical frame reparsed document")
	}
}

func TestStreamingOpenFenceOnlyReparsesNewContentLines(t *testing.T) {
	var state streamDocument
	source := "Before\n\n```go\n"
	for index := range 240 {
		source += fmt.Sprintf("fmt.Println(%d)\n", index)
		blocks := state.update(source, true)
		last := blocks[len(blocks)-1]
		_, code := codeBlockSource(last.node, last.source)
		if !strings.Contains(code, fmt.Sprintf("fmt.Println(%d)", index)) || strings.Count(code, "fmt.Println") != index+1 {
			t.Fatalf("fence lost appended lines at %d: %q", index, code)
		}
	}
	if state.parsedBytes > len(source)*8 {
		t.Fatalf("open fence reparsed complete content: parsed=%d source=%d", state.parsedBytes, len(source))
	}
	t.Logf("240 appended fence lines: source=%d grammar-input=%d", len(source), state.parsedBytes)
	source += "```\n\nAfter"
	settled := state.update(source, false)
	if got, want := parsedBlockText(settled, false), concat(Spans(source, Style{})); got != want {
		t.Fatalf("closed fence=%q full=%q", got, want)
	}
}

func TestStreamingFenceSplitNewlineAndClosingMarkerMatchesFullGrammar(t *testing.T) {
	for _, source := range []string{"   ~~~~go\r\n   one\r\n\t two\r\n   ~~~~~\r\nAfter", "```txt\nalpha\n\nbeta\n```\nAfter", "Title\n=====\n\nOne\n\nTwo\n\nThree"} {
		var state streamDocument
		for end := 1; end <= len(source); end++ {
			blocks := state.update(source[:end], true)
			if got, want := parsedBlockText(blocks, true), fullParsedText(source[:end], true); got != want {
				t.Fatalf("split %d of %q: incremental=%q full=%q", end, source, got, want)
			}
		}
	}
}

func TestStreamingCodeWrapChoiceAndFenceOffsetSurviveFreezing(t *testing.T) {
	source := "```go\nfmt.Println(1)\n```\n\nTail"
	streaming := true
	var offsets []int
	st := Style{DocumentLayout: true, RenderFence: func(_ *ui.Context, _, _ string, offset int) bool { offsets = append(offsets, offset); return false }}
	tt := ui.NewTester(func(c *ui.Context) {
		st.Streaming = streaming
		ui.Column(c).Fill().Children(func() { View(c, source, st) })
	}, 500, 700)
	if err := tt.Click(codeWrapLabelsFirst(st)); err != nil {
		t.Fatal(err)
	}
	source += "\n\nAnother\n\nLast"
	tt.Frame()
	streaming = false
	tt.Frame()
	_, noWrap := codeWrapLabels(st)
	if _, ok := tt.Find(noWrap); !ok {
		t.Fatalf("freezing or settlement reset code wrap: %q", tt.Texts())
	}
	for _, offset := range offsets {
		if offset != 6 {
			t.Fatalf("fence absolute offset changed: %v", offsets)
		}
	}
}

func codeWrapLabelsFirst(st Style) string { wrap, _ := codeWrapLabels(st); return wrap }

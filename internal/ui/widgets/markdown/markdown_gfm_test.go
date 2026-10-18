package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

func TestGFMTaskListAndStrikethrough(t *testing.T) {
	spans := Spans("- [x] ~~done~~\n- [ ] ~~`pending.go`~~", Style{FontSize: 14})
	if got := concat(spans); got != "• ☑ done\n• ☐ pending.go" {
		t.Fatalf("task list = %q", got)
	}
	for _, text := range []string{"done", "pending.go"} {
		found := false
		for _, span := range spans {
			if span.Text == text && span.Strikethrough {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing strikethrough on %q: %#v", text, spans)
		}
	}
}

func TestGFMTableAndCodeBlockOrder(t *testing.T) {
	source := "Before\n\n| Name | Value |\n| --- | ---: |\n| **甲** | `42` |\n\nBetween\n\n```go\nfmt.Println(\"ok\")\n\n```\n\nAfter"
	var blocks []richBlock
	spans := Spans(source, Style{FontSize: 14, blockAt: func(start, end int, node ast.Node) {
		blocks = append(blocks, richBlock{start, end, node})
	}})
	if len(blocks) != 2 {
		t.Fatalf("blocks = %#v", blocks)
	}
	if _, ok := blocks[0].node.(*extast.Table); !ok {
		t.Fatalf("first block = %T", blocks[0].node)
	}
	if _, ok := blocks[1].node.(*ast.FencedCodeBlock); !ok {
		t.Fatalf("second block = %T", blocks[1].node)
	}
	if got := concat(spans[blocks[0].start:blocks[0].end]); got != "Name | Value\n甲 | 42\n" {
		t.Fatalf("table fallback = %q", got)
	}
	if got := concat(spans[blocks[0].end:blocks[1].start]); got != "Between\n" {
		t.Fatalf("intervening prose = %q", got)
	}
	language, code := codeBlockSource(blocks[1].node, source)
	if language != "go" || code != "fmt.Println(\"ok\")\n\n" {
		t.Fatalf("clipboard source lost exact bytes: %q %q", language, code)
	}
	if got := concat(spans[blocks[1].end:]); got != "After" {
		t.Fatalf("trailing prose = %q", got)
	}
}

func TestGFMInlineFileOffsetsAndAutolinks(t *testing.T) {
	var targets = map[int]string{}
	spans := Spans("```text\nblock\n```\n\n~~`file.go`~~ https://example.com", Style{codeSpanAt: func(index int, value string) {
		targets[index] = value
	}})
	if len(targets) != 1 {
		t.Fatalf("file targets = %#v", targets)
	}
	for index, value := range targets {
		if spans[index].Text != value || !spans[index].Strikethrough {
			t.Fatalf("target %d = %#v", index, spans[index])
		}
	}
	if !strings.Contains(concat(spans), "https://example.com") {
		t.Fatal("bare link disappeared")
	}
}

// Compile-time coverage for the native block rendering entry point.
var _ func(*ui.Context, string, Style, []ui.Span, map[int]string, []richBlock) = viewBlocks

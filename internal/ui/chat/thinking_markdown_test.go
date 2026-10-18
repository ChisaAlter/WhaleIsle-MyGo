package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestThinkingMarkdownDisclosureStreamingAndCopy(t *testing.T) {
	body := "  **reasoning**\n\n```go\nprintln(1)\n```\n\n" + strings.Repeat("tail ", 100) + "END  "
	expanded := map[string]bool{}
	tt := ui.NewTester(func(c *ui.Context) { ThinkingBlockView(c, "reasoning", body, &expanded, ThinkingStyle{Size: 12}) }, 800, 1000)
	if _, ok := tt.Find("Copy code"); ok {
		t.Fatal("collapsed reasoning exposed full markdown")
	}
	if err := tt.Click("▸ " + i18n.T(i18n.KeyChatThinking)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Copy code"); !ok {
		t.Fatal("expanded reasoning is not shared Markdown")
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "println(1)\n" {
		t.Fatalf("code clipboard %q", tt.Clipboard())
	}
	if err := tt.Click("Copy reasoning reasoning"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != body {
		t.Fatal("full reasoning copy lost raw syntax, whitespace or tail")
	}
	body = strings.Replace(body, "println(1)", "println(2)", 1) + " streaming update"
	tt.Frame()
	tt.Click("Copy code")
	if tt.Clipboard() != "println(2)\n" {
		t.Fatal("stream update stuck on previous render")
	}
	tt.Click("Copy reasoning reasoning")
	if tt.Clipboard() != body {
		t.Fatal("full copy stale after stream update")
	}
	tt.Click("▾ " + i18n.T(i18n.KeyChatThinking))
	if expanded["reasoning"] {
		t.Fatal("collapse state lost")
	}
	if _, ok := tt.Find("Copy code"); ok {
		t.Fatal("collapsed code remained visible")
	}
}

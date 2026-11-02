package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func markdownCopyTester(source *string, streaming *bool) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, *source, Style{DocumentLayout: true, FontSize: 14, Streaming: *streaming,
				CopyLabel: "Copy source", CopiedLabel: "Source copied", WrapLabel: "Wrap source", NoWrapLabel: "Keep columns"})
		})
	}, 340, 700)
}

func assertMarkdownCodeCopy(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	tt.SetClipboard("untouched before activation")
	tt.Frame()
	if tt.Clipboard() != "untouched before activation" {
		t.Fatal("painting code wrote the clipboard")
	}
	if err := tt.Click("Copy source"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != want {
		t.Fatalf("copied source changed: got %q want %q", got, want)
	}
	if _, ok := tt.Find("Source copied"); !ok {
		t.Fatal("copy activation did not show confirmation")
	}
	if _, ok := tt.Find("Copy source"); ok {
		t.Fatal("copy action remained active during confirmation")
	}
	tt.SetClipboard("keep during confirmation")
	if err := tt.Click("Source copied"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.Clipboard() != "keep during confirmation" {
		t.Fatal("confirmation or a later paint repeated the copy")
	}
}

func TestMarkdownCodeCopyPreservesTabsAndExistingSourceEndings(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"makefile", "```makefile\nbuild:\n\t@echo done\n```", "build:\n\t@echo done\n"},
		{"python", "```python\nif ready:\n\tif nested:\n\t\tprint('鲸鱼')\n```", "if ready:\n\tif nested:\n\t\tprint('鲸鱼')\n"},
		{"tsv-multibyte", "```text\n名称\t数值\t🙂\n鲸鱼\t42\t完成\n```", "名称\t数值\t🙂\n鲸鱼\t42\t完成\n"},
		{"empty", "```text\n```", ""},
		{"blank-line", "```text\n\n```", "\n"},
		{"trailing-blank-lines", "```text\nx\tend\n\n\n```", "x\tend\n\n\n"},
		{"unterminated-last-line", "```text\nlast\tvalue", "last\tvalue\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, streaming := tc.source, false
			tt := markdownCopyTester(&source, &streaming)
			if strings.Contains(tc.want, "\t") {
				for _, text := range tt.Texts() {
					if strings.Contains(text, "\t") {
						t.Fatalf("display stopped expanding tab indentation: %q", text)
					}
				}
				if !tt.HasText("    ") {
					t.Fatal("expanded tab indentation disappeared from the display")
				}
			}
			assertMarkdownCodeCopy(t, tt, tc.want)
		})
	}
}

func TestMarkdownCodeCopyAfterWrapAndStreamingSettlement(t *testing.T) {
	for _, stage := range []string{"wrap", "streaming", "settled"} {
		t.Run(stage, func(t *testing.T) {
			code := "\tprint('" + strings.Repeat("鲸鱼 value ", 12) + "')\n"
			source, streaming := "```python\n"+code, true
			tt := markdownCopyTester(&source, &streaming)
			before, ok := tt.Find(strings.ReplaceAll(strings.TrimRight(code, "\n"), "\t", "    "))
			if !ok {
				t.Fatalf("streaming code body missing: %q", tt.Texts())
			}
			if err := tt.Click("Wrap source"); err != nil {
				t.Fatal(err)
			}
			after, ok := tt.Find(strings.ReplaceAll(strings.TrimRight(code, "\n"), "\t", "    "))
			if !ok || after.H <= before.H {
				t.Fatalf("wrap did not change visible code geometry: before=%+v after=%+v", before, after)
			}
			if stage != "wrap" {
				code += "\t\tprint('next\t🙂')\n\n"
				source = "```python\n" + code
				tt.Frame()
			}
			if stage == "settled" {
				source += "```\n\nTail\n\nMore\n\nLast"
				streaming = false
				tt.Frame()
			}
			tt.Frame()
			if _, ok := tt.Find("Keep columns"); !ok {
				t.Fatal("code append/settlement reset the user's wrap choice")
			}
			assertMarkdownCodeCopy(t, tt, code)
		})
	}
}

func TestMarkdownCodeCopyKeepsFenceOwnerInput(t *testing.T) {
	source := "```owned\n\tprogram\n\n```"
	var received string
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			View(c, source, Style{DocumentLayout: true, RenderFence: func(c *ui.Context, language, program string, offset int) bool {
				if language != "owned" || offset != len("```owned\n") {
					t.Fatalf("fence owner coordinates changed: %q %d", language, offset)
				}
				received = program
				ui.Text(c, "Owned fence")
				return true
			}})
		})
	}, 340, 300)
	if received != "    program\n\n" || !tt.HasText("Owned fence") {
		t.Fatalf("existing fence owner input changed: %q", received)
	}
	if _, ok := tt.Find("Copy code"); ok {
		t.Fatal("owned fence gained a generic copy action")
	}
}

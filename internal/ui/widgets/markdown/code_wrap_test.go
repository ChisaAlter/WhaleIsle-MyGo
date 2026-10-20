package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestCodeWrapDefaultIsCallerScopedAndToggleRetained(t *testing.T) {
	for _, initial := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Children(func() { View(c, "```go\nfmt.Println(1)\n```", Style{CodeWrapDefault: initial, WrapLabel: i18n.T(i18n.KeyMarkdownWrap), NoWrapLabel: i18n.T(i18n.KeyMarkdownNoWrap)}) })
		}, 400, 300)
		// The wrap seat reports the action it performs: a wrapped block offers
		// "保留列宽", an unwrapped one offers "自动换行".
		first, after := i18n.T(i18n.KeyMarkdownWrap), i18n.T(i18n.KeyMarkdownNoWrap)
		if initial {
			first, after = after, first
		}
		if err := tt.Click(first); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if _, ok := tt.Find(after); !ok {
			t.Fatalf("initial=%v: default overrode the user's retained toggle", initial)
		}
	}
}

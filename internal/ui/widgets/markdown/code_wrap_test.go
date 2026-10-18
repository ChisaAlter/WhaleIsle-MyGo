package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCodeWrapDefaultIsCallerScopedAndToggleRetained(t *testing.T) {
	for _, initial := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Children(func() { View(c, "```go\nfmt.Println(1)\n```", Style{CodeWrapDefault: initial}) })
		}, 400, 300)
		first, after := "Wrap lines", "Unwrap lines"
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

package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// terminalBlock must paint one tinted seat in either theme, hold the command and
// the output, and scroll long output inside the seat instead of growing it.
func TestTerminalBlockSeatInBothThemes(t *testing.T) {
	long := strings.Join([]string{
		"ok  	whaleisle/internal/ui/chat	2.41s",
		"ok  	whaleisle/internal/ui/frame	9.02s",
		strings.Repeat("wide   ", 40),
	}, "\n")
	for _, tc := range []struct {
		name  string
		id    string
		dark  bool
		field ui.Color
	}{
		{name: "light", id: "deepseek.light", field: chrome.Color(theme.Resolve("deepseek").Light.Field)},
		{name: "dark", id: "deepseek", dark: true, field: chrome.Color(theme.Resolve("deepseek").Dark.Field)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				if tc.dark {
					c.SetTheme(ui.DarkTheme())
				}
				colors := theme.Resolve(tc.id)
				active := colors.Active(tc.dark)
				ui.Column(c).FillWidth().Padding(10).Children(func() {
					terminalBlock(c, terminalBlockStyle{
						FG:    chrome.Color(active.FG),
						Muted: chrome.Color(active.Muted),
						Field: tc.field,
						Size:  11,
					}, "call-1", "go test ./internal/ui/chat -count=1", long, func(string) {})
				})
			}, 640, 520)
			seat, ok := tt.Find("Terminal output")
			if !ok {
				t.Fatalf("%s terminal seat missing: %q", tc.name, tt.Texts())
			}
			if !tt.HasText("go test ./internal/ui/chat -count=1") {
				t.Fatalf("%s command missing from the seat", tc.name)
			}
			if !tt.HasText("ok  	whaleisle/internal/ui/chat	2.41s") {
				t.Fatalf("%s output missing from the seat", tc.name)
			}
			if seat.H > terminalMaxHeight+3*terminalLineHeight {
				t.Fatalf("%s seat grew past the scroll cap: %v", tc.name, seat.H)
			}
			img := tt.Image()
			got := img.RGBAAt(int(seat.X)+3, int(seat.Y)+3)
			if got.R != tc.field.R || got.G != tc.field.G || got.B != tc.field.B {
				t.Fatalf("%s seat fill = %v, want the field color %v", tc.name, got, tc.field)
			}
		})
	}
}

// A one-line call keeps a compact seat; the copy action yields command + output.
func TestTerminalBlockCopyPayloadAndCompactSeat(t *testing.T) {
	var copied string
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(10).Children(func() {
			terminalBlock(c, terminalBlockStyle{FG: ui.RGB(0, 0, 0), Muted: ui.RGB(110, 110, 110), Field: ui.RGB(245, 245, 245)},
				"call-2", "printf hi", "hi", func(text string) { copied = text })
		})
	}, 640, 240)
	seat, ok := tt.Find("Terminal output")
	if !ok {
		t.Fatal("terminal seat missing")
	}
	if seat.H > 2*terminalLineHeight+3*terminalPad+terminalCopySize {
		t.Fatalf("one-line seat too tall: %v", seat.H)
	}
	if err := tt.Click(i18n.T(i18n.KeyChatToolCopy)); err != nil {
		t.Fatal(err)
	}
	if copied != "printf hi\nhi" {
		t.Fatalf("copy payload = %q", copied)
	}
}

func i18n_ToolCopy() string { return copyLabelForTest }

var copyLabelForTest = "复制"

package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

// A button with its own background skips styleButton's hover, so the composer
// paints hover and pressed itself; this pins that behavior down.
func TestComposerPrimaryAndAddPaintHoverAndPressed(t *testing.T) {
	d := NewDraftState("hello")
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true,
		SuggestList: &ui.ListState{Selected: new(int)},
		Snapshot:    session.ViewState{SelectedID: "s", ClientReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 300)
	colors := testColors()
	primary, ok := tt.Find(i18n.T(i18n.KeyComposerSend))
	if !ok {
		t.Fatal("send button missing")
	}
	// Sample a point left of the arrow glyph inside the 34px disc.
	point := func(r ui.Rect) (int, int) { return int(r.X + 10), int(r.Y + r.H/2) }
	read := func() ui.Color {
		x, y := point(primary)
		pixel := tt.Image().RGBAAt(x, y)
		return ui.RGBA(pixel.R, pixel.G, pixel.B, 255)
	}
	base := read()
	if !sameColor(base, chrome.Color(colors.ButtonInfoFill)) {
		t.Fatalf("send resting fill %v, want %v", base, chrome.Color(colors.ButtonInfoFill))
	}
	tt.Move(primary.X+primary.W/2, primary.Y+primary.H/2)
	tt.Frame()
	if hover := read(); sameColor(hover, base) {
		t.Fatal("send button does not react to hover")
	}
	tt.Press(primary.X+primary.W/2, primary.Y+primary.H/2)
	tt.Frame()
	pressed := read()
	if sameColor(pressed, base) {
		t.Fatal("send button does not react to press")
	}
	tt.Release(primary.X+primary.W/2, primary.Y+primary.H/2)
	tt.Move(0, 0)
	tt.Frame()
	if restore := read(); !sameColor(restore, base) {
		t.Fatalf("send fill did not return to rest: %v", restore)
	}
	// The attach disc uses the neutral selector fill with the same rule.
	add, ok := tt.Find("+")
	if !ok {
		t.Fatal("attach button missing")
	}
	tt.Move(add.X+add.W/2, add.Y+add.H/2)
	tt.Frame()
	selector := chrome.Color(colors.SelectorFill)
	x, y := int(add.X+add.W/2), int(add.Y+add.H/2)
	pixel := tt.Image().RGBAAt(x, y)
	if sameColor(ui.RGBA(pixel.R, pixel.G, pixel.B, 255), selector) {
		t.Fatal("attach button does not react to hover")
	}
}

func sameColor(a, b ui.Color) bool {
	const tolerance = 6
	delta := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return delta(a.R, b.R) <= tolerance && delta(a.G, b.G) <= tolerance && delta(a.B, b.B) <= tolerance && a.A == b.A
}

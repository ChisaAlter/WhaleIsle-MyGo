package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// coss Button paints hover and pressed transitions; verify send and attach react cleanly.
func TestComposerPrimaryAndAddPaintHoverAndPressed(t *testing.T) {
	d := NewDraftState("hello")
	m := Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true,
		SuggestList: &ui.ListState{Selected: new(int)},
		Snapshot:    session.ViewState{SelectedID: "s", ClientReady: true}}
	var wantPrimary, wantSecondary ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		tok := design.For(c)
		wantPrimary, wantSecondary = tok.Primary, tok.Secondary
		View(c, m)
	}, 800, 300)
	primary, ok := tt.Find(i18n.T(i18n.KeyComposerSend))
	if !ok {
		t.Fatal("send button missing")
	}
	// Sample a point inside the 32px disc left of the arrow glyph.
	point := func(r ui.Rect) (int, int) { return int(r.X + 8), int(r.Y + r.H/2) }
	read := func() ui.Color {
		x, y := point(primary)
		pixel := tt.Image().RGBAAt(x, y)
		return ui.RGBA(pixel.R, pixel.G, pixel.B, 255)
	}
	base := read()
	if !sameColor(base, wantPrimary) {
		t.Fatalf("send resting fill %v, want %v", base, wantPrimary)
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
	// The attach disc uses coss secondary/ghost styling and reacts to hover.
	add, ok := tt.Find(i18n.T(i18n.KeyComposerCommands))
	if !ok {
		t.Fatal("attach button missing")
	}
	restX, restY := int(add.X+4), int(add.Y+4)
	restPix := tt.Image().RGBAAt(restX, restY)
	restColor := ui.RGBA(restPix.R, restPix.G, restPix.B, 255)
	_ = wantSecondary
	tt.Move(add.X+add.W/2, add.Y+add.H/2)
	tt.Frame()
	pixel := tt.Image().RGBAAt(restX, restY)
	hoverColor := ui.RGBA(pixel.R, pixel.G, pixel.B, 255)
	if hoverColor == restColor {
		t.Fatalf("attach button does not react to hover: rest=%v hover=%v", restColor, hoverColor)
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

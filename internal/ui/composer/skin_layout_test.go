package composer

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

func TestSkinLayoutComposerUsesInstalledSurfaceWithoutChangingDraftGeometry(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old) })
	p := theme.Defaults()
	p.Light, p.Dark = "codex", "claude"
	theme.SetPreferences(p)
	for _, dark := range []bool{false, true} {
		d := NewDraftState("")
		var tok *design.Tokens
		tt := ui.NewTester(func(c *ui.Context) {
			if dark {
				c.SetTheme(ui.DarkTheme())
			}
			tok = design.For(c)
			ui.Column(c).FillWidth().Background(tok.Background).Children(func() {
				View(c, Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: true, Hero: true, ThemeID: theme.IDSystem, SystemDark: dark})
			})
		}, 800, 220)
		r, ok := tt.Find("Composer card")
		if !ok {
			t.Fatal("card missing")
		}
		input, _ := tt.Find("Composer draft")
		if math.Abs(float64(r.H-114)) > .1 || math.Abs(float64(input.H-52)) > .1 || input.Y-r.Y != 8 {
			t.Fatalf("dark%v sourcecardgeometry card%+v draft%+v", dark, r, input)
		}
		fill := composerCardFill(tok)
		bg := tok.Background
		got := tt.Image().RGBAAt(int(r.X+r.W/2), int(r.Y+16))
		for i, pair := range [][3]uint8{{got.R, fill.R, bg.R}, {got.G, fill.G, bg.G}, {got.B, fill.B, bg.B}} {
			expected := uint8((int(pair[1])*int(fill.A) + int(pair[2])*(255-int(fill.A)) + 127) / 255)
			if absByte(pair[0], expected) > 2 {
				t.Fatalf("dark%v installed surface channel %d=%d want %d (pixel=%v fill=%v)", dark, i, pair[0], expected, got, fill)
			}
		}
	}
}

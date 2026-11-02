package composer

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
	"whaleisle/internal/ui/chrome"
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
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Children(func() {
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
		expected := chrome.Color(theme.ResolvedSkinLayout(theme.IDSystem, dark).ComposerBackground)
		got := tt.Image().RGBAAt(int(r.X+r.W/2), int(r.Y+4))
		if got.R != expected.R || got.G != expected.G || got.B != expected.B || got.A != 255 {
			t.Fatalf("dark%v installed surface pixel=%v want%v", dark, got, expected)
		}
	}
}

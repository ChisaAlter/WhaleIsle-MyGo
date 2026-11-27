package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)

func TestComposerFocusBorderSoftAndInvariant(t *testing.T) {
	for _, tc := range []struct {
		name      string
		scheme    design.Scheme
		ring      ui.Color
		wantAlpha uint8
	}{
		{"light-opaque-ring", design.Light, ui.RGB(65, 118, 230), 140},
		{"light-translucent-ring", design.Light, ui.RGBA(65, 118, 230, 0.35), 140},
		{"dark-opaque-ring", design.Dark, ui.RGB(122, 170, 255), 158},
		{"dark-translucent-ring", design.Dark, ui.RGBA(122, 170, 255, 0.35), 158},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tok := design.Resolve(tc.scheme, ui.RGB(65, 118, 230), ui.Color{})
			tok.Ring = tc.ring
			got := composerCardBorder(&tok, true)
			if got.R != tc.ring.R || got.G != tc.ring.G || got.B != tc.ring.B {
				t.Fatalf("focused border RGB=%v want RGB from %v", got, tc.ring)
			}
			if got.A < tc.wantAlpha-2 || got.A > tc.wantAlpha+2 {
				t.Fatalf("focused border alpha=%d want ~%d (soft translucent accent, never opaque 255)", got.A, tc.wantAlpha)
			}
			resting := composerCardBorder(&tok, false)
			if resting != tok.Input {
				t.Fatalf("resting border=%v want t.Input=%v", resting, tok.Input)
			}
		})
	}
}

func TestComposerFocusZeroShiftAndRefinedHalo(t *testing.T) {
	for _, dark := range []bool{false, true} {
		d := NewDraftState("")
		tt := ui.NewTester(func(c *ui.Context) {
			if dark {
				c.SetTheme(ui.DarkTheme())
			} else {
				c.SetTheme(ui.LightTheme())
			}
			ui.Column(c).Fill().Background(design.For(c).Background).Padding(24).Children(func() {
				View(c, Model{
					State:      d,
					Draft:      &d.Text,
					SystemDark: dark,
				})
			})
		}, 520, 220)
		if dark {
			tt.SetDark(true)
		}
		tt.Frame()
		tt.Frame()
		ui.AssertHoverFocusZeroShift(t, tt)
	}
}

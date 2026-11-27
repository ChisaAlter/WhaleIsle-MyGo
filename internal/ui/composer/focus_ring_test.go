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
		accent    ui.Color
		ring      ui.Color
		wantAlpha uint8
	}{
		{"light-deepseek-opaque-ring", design.Light, ui.RGB(65, 118, 230), ui.RGB(65, 118, 230), 242},
		{"light-deepseek-translucent-ring", design.Light, ui.RGB(65, 118, 230), ui.RGBA(65, 118, 230, 0.35), 242},
		{"dark-deepseek-opaque-ring", design.Dark, ui.RGB(122, 170, 255), ui.RGB(122, 170, 255), 224},
		{"dark-deepseek-translucent-ring", design.Dark, ui.RGB(122, 170, 255), ui.RGBA(122, 170, 255, 0.35), 224},
		{"dark-neutral-gray", design.Dark, ui.Hex("#52525b"), ui.Color{}, 224},
		{"light-neutral-gray", design.Light, ui.Hex("#52525b"), ui.Color{}, 242},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tok := design.Resolve(tc.scheme, tc.accent, ui.Color{})
			if tc.ring.A > 0 {
				tok.Ring = tc.ring
			}
			got := composerCardBorder(&tok, true)
			if tc.ring.A > 0 && (got.R != tc.ring.R || got.G != tc.ring.G || got.B != tc.ring.B) {
				t.Fatalf("focused border RGB=%v want RGB from %v", got, tc.ring)
			}
			if got.A < tc.wantAlpha-2 || got.A > tc.wantAlpha+2 {
				t.Fatalf("focused border alpha=%d want ~%d", got.A, tc.wantAlpha)
			}
			bg := design.Base(tc.scheme).Background
			if c := design.Contrast(got.Over(bg), bg); c < 3.0 {
				t.Fatalf("focused border %v contrast on %v = %.2f < 3.0", got, bg, c)
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

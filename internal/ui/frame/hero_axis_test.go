package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// TestHeroComposerCardUsesTheNativeColumnAxis pins the round 210 fix: the width
// formulas are percentages of the native column (viewport − sidebar), not of the
// divider-shrunk Sizes.Center, so a 1440-wide window with a 280 sidebar yields the
// installed card's 774.39 instead of 773.3.
func TestHeroComposerCardUsesTheNativeColumnAxis(t *testing.T) {
	cases := []struct {
		name    string
		width   int
		center  float32
		sidebar float32
		right   float32
		want    float32
	}{
		// 0.64 × (1440 − 280) = 742.4 → seat 806.4 → card 774.4.
		{"1440 no dock", 1440, 1158, 280, 0, 774.39},
		// With a right dock the drawn column is authoritative: 0.64 × 1158.
		{"1440 with dock", 1440, 1158, 280, 320, 741.12 + 64 - 32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sizes := LayoutSizes{Center: tc.center, Sidebar: tc.sidebar, Right: tc.right}
			colors := theme.Resolve("deepseek.dark").Dark
			var card ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: session.New(), Sizes: &sizes, ThemeID: "deepseek.dark", SystemDark: true}
				ui.Column(c).Fill().Children(func() {
					mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, tc.width, 860)
			tt.Frame()
			if r, ok := tt.Find("Composer card"); ok {
				card = r
			}
			if card.W == 0 {
				t.Fatalf("composer card not found at %d wide", tc.width)
			}
			if math.Abs(float64(card.W-tc.want)) > 0.6 {
				t.Fatalf("card width=%.2f, want %.2f (axis must use the native column)", card.W, tc.want)
			}
		})
	}
}

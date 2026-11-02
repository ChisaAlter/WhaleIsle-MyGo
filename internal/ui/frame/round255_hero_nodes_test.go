package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Every headline child must stay inside its source wrap envelope, including a
// narrow column. Mark identity and label survive the updated source badge.
func TestRound255HeroRowNodes(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, width := range []int{390, 1440} {
		tt := ui.NewTester(func(c *ui.Context) {
			conversationHeroHeadline(c, Model{ThemeID: theme.IDDeepseekLight}, theme.Resolve(theme.IDDeepseekLight).Light)
		}, width, 160)
		for _, name := range []string{"Conversation hero fish", i18n.T(i18n.KeyChatHeroSlogan), i18n.T(i18n.KeyChatHeroPreview)} {
			r, ok := tt.Find(name)
			if !ok {
				t.Fatalf("missing%q", name)
			}
			if r.X < 0 || r.Y < 0 || r.X+r.W > float32(width)+.1 || r.Y+r.H > 160 {
				t.Fatalf("%d: %q clipped=%+v", width, name, r)
			}
			if name == "Conversation hero fish" && (r.W != 34 || r.H != 34) {
				t.Fatalf("official34mark=%+v", r)
			}
		}
	}
}

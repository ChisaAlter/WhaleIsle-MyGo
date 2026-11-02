package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// HeroShell.stack contributes its empty-body 12-DIP gap, followed by the
// ConversationRoot composerStack's 8-DIP gap. The official mark sets a 34-DIP
// row around the 32-DIP slogan, independent of the obsolete 250-DIP seat.
func TestHeroHeadlineCarriesTheNativeStackHeight(t *testing.T) {
	if conversationHeroHeadlineFoot != 20 || conversationHeroHeadlineHeight != 34 {
		t.Fatalf("headline source geometry=%.2f+%.2f, want34+20", conversationHeroHeadlineHeight, conversationHeroHeadlineFoot)
	}
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	colors := theme.Resolve("deepseek.dark").Dark
	tt := ui.NewTester(func(c *ui.Context) {
		conversationHeroHeadline(c, Model{ThemeID: "deepseek.dark", SystemDark: true}, colors)
	}, 1440, 864)
	tt.Frame()
	headline, ok := tt.Find("Conversation hero headline")
	if !ok {
		t.Fatal("headline layout node missing")
	}
	if math.Abs(float64(headline.H-54)) > .1 {
		t.Fatalf("headline plus source gap height=%.2f want54", headline.H)
	}
	slogan, ok := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
	if !ok {
		t.Fatal("slogan missing")
	}
	if math.Abs(float64(slogan.H-32)) > .1 {
		t.Fatalf("slogan line box=%.2f want32", slogan.H)
	}
}

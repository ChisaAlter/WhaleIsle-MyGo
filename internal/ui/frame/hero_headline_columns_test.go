package frame

import (
	"math"
	"testing"
	"whaleisle/internal/ui/i18n"
)

// Installed source gap10 between mark and titleGroup; gap7 inside titleGroup.
func TestHeroHeadlineMatchesTheNativeColumns(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := chatChromeTester(t)
	slogan, ok := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
	if !ok {
		t.Fatal("slogan missing")
	}
	badge, _ := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
	fish, _ := tt.Find("Conversation hero fish")
	if math.Abs(float64(slogan.X-(fish.X+fish.W)-10)) > .1 || math.Abs(float64(badge.X-(slogan.X+slogan.W)-7)) > .1 {
		t.Fatalf("sourcegaps fish=%+v slogan=%+v badge=%+v", fish, slogan, badge)
	}
	card, _ := tt.Find("Composer card")
	if math.Abs(float64((fish.X+badge.X+badge.W)/2-(card.X+card.W/2))) > .1 {
		t.Fatalf("headline group is not centered over composer")
	}
}

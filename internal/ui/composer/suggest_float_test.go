package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestSuggestPopupFloatsOutsideCard(t *testing.T) {
	_, m, _ := suggestionFixture("@")
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	card, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card missing")
	}
	menu, ok := tt.Find(i18n.T(i18n.KeyComposerMentionMenuTitle))
	if !ok {
		t.Fatalf("suggest menu missing; texts=%q", tt.Texts())
	}
	if menu.Y < card.Y+card.H && menu.Y+menu.H > card.Y {
		t.Fatalf("menu %+v overlaps card at %+v", menu, card)
	}
	if menu.W > card.W+1 {
		t.Fatalf("menu width %v exceeds card %v", menu.W, card.W)
	}
}
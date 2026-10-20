package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestDeveloperPreferenceHidesChangesKeepsDeliveries(t *testing.T) {
	d := session.TurnDelivery{Turn: 1, Changes: &session.TurnFileTarget{Turn: 1, Seq: 3}, Files: []session.TurnPresentedFile{{Path: "report.txt"}}}
	tt := ui.NewTester(func(c *ui.Context) { TurnDeliveryView(c, d, Model{DisableDeveloperTools: true}, ToolCardStyle{}) }, 800, 600)
	if _, ok := tt.Find(fmt.Sprintf(i18n.T(i18n.KeyChatDeliveryReview), 1, 3)); ok {
		t.Fatal("developer-only change action visible")
	}
	if _, ok := tt.Find("report.txt"); !ok {
		t.Fatal("ordinary delivery hidden")
	}
	if d.Changes == nil {
		t.Fatal("projection changed original delivery")
	}
}

package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestDeveloperPreferenceReturnsTrajectoryToChat(t *testing.T) {
	tab := "trajectory"
	sizes := DefaultLayout()
	sizes.Bottom = 0
	_ = ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, ChatPaneTab: &tab, DisableDeveloperTools: true})
	}, 1200, 800)
	if tab != "chat" {
		t.Fatal("disabled trajectory remains selected")
	}
}

package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestCustomPanelsUseActualDockResource(t *testing.T) {
	tab := "custom-panels"
	sizes := DefaultLayout()
	calls := 0
	model := Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, RightPaneTab: &tab, CustomPanelContent: func(c *ui.Context) { calls++; ui.Text(c, "Bot panel content") }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 1200, 800)
	if calls == 0 || !tt.HasText("Bot panel content") {
		t.Fatal("custom panels content did not render")
	}
	w := sizes.Workspace
	n := w.Root.find(w.Active)
	if n == nil || n.Kind != "custom-panels" {
		t.Fatal("no dock resource")
	}
	id := n.ID
	w.selectKind(n, "files")
	w.selectKind(w.Root.find(w.Active), "custom-panels")
	if w.Active != id {
		t.Fatal("navigation must reuse the Panel library resource")
	}
}

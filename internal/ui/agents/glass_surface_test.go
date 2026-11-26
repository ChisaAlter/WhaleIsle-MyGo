package agents

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
)

func TestAgentsPanelSingleLayerTransparentOverGlass(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	design.SetGlass(true, func(_ design.Scheme, _ *design.Palette) {})

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(18, 120, 210)).Children(func() {
			View(c, Model{Snapshot: session.ViewState{SelectedID: "s1"}})
		})
	}, 320, 240)
	tt.Frame()
	got := tt.Image().RGBAAt(20, 210)
	if got.R != 18 || got.G != 120 || got.B != 210 {
		t.Fatalf("agents root painted nested fill %+v over canvas", got)
	}
}

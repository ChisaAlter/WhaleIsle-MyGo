package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)

func TestComposerCardNoOuterShadowAnd12pxMenuRadius(t *testing.T) {
	defer design.SetGlass(false, nil)
	for _, glass := range []bool{false, true} {
		design.SetGlass(glass, func(_ design.Scheme, p *design.Palette) {
			p.Card = p.Card.Alpha(0.80)
			p.Popover = p.Popover.Alpha(0.70)
		})
		var menuBounds ui.Rect
		tt := ui.NewTester(func(c *ui.Context) {
			t := design.For(c)
			card := ui.Box(c).Width(200).Height(60)
			composerCardElevation(card, t, false, false)
			menuBounds = menuPanelLook(c, ui.Box(c).Width(160).Height(80)).Bounds()
		}, 300, 200)
		tt.Frame()
		if menuBounds.W != 160 {
			t.Fatalf("menuPanelLook width=%v, want 160", menuBounds.W)
		}
	}
}

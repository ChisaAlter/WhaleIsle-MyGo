package files

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)

func TestFilesPanelTransparentUnderPureDesignGlass(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, func(_ design.Scheme, _ *design.Palette) {})

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(32, 112, 192)).Children(func() {
			View(c, Model{})
		})
	}, 320, 240)
	tt.Frame()
	if got := tt.Image().RGBAAt(40, 180); got.R != 32 || got.G != 112 || got.B != 192 {
		t.Fatalf("files panel under pure design.Glass painted opaque fill: %+v", got)
	}
}

package diffsurface

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
)

func TestDiffPanelTransparentUnderPureDesignGlass(t *testing.T) {
	defer design.SetGlass(false, nil)
	design.SetGlass(true, func(_ design.Scheme, _ *design.Palette) {})

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(24, 104, 184)).Children(func() {
			View(c, Model{})
		})
	}, 320, 240)
	tt.Frame()
	if got := tt.Image().RGBAAt(40, 180); got.R != 24 || got.G != 104 || got.B != 184 {
		t.Fatalf("diff panel under pure design.Glass painted opaque fill: %+v", got)
	}
}

package design

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDrawHairlineConstantPhysicalThicknessAcrossScalesAndParity(t *testing.T) {
	scales := []struct {
		scale  float32
		wantPx int
	}{
		{1.0, 1},
		{1.25, 1},
		{1.5, 1},
		{1.75, 1},
		{2.0, 2},
	}
	positions := []float32{10, 11, 12, 13, 20, 21, 30, 31}

	for _, sc := range scales {
		for _, pos := range positions {
			// Horizontal hairline
			ttH := ui.NewTester(func(c *ui.Context) {
				ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
					ui.Box(c).Absolute().Left(0).Top(pos).Width(40).Height(1).Draw(func(p *ui.Painter, r ui.Rect) {
						DrawHairline(p, r, false, ui.RGB(0, 0, 0))
					})
				})
			}, 40, 50)
			ttH.SetScale(sc.scale)
			ttH.Frame()
			img := ttH.Image()
			darkRows := 0
			for y := 0; y < img.Bounds().Dy(); y++ {
				if img.RGBAAt(20, y).R < 128 {
					darkRows++
				}
			}
			if darkRows != sc.wantPx {
				t.Fatalf("horizontal scale=%g pos=%g: painted %d physical px rows, want %d", sc.scale, pos, darkRows, sc.wantPx)
			}

			// Vertical hairline
			ttV := ui.NewTester(func(c *ui.Context) {
				ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
					ui.Box(c).Absolute().Left(pos).Top(0).Width(1).Height(40).Draw(func(p *ui.Painter, r ui.Rect) {
						DrawHairline(p, r, true, ui.RGB(0, 0, 0))
					})
				})
			}, 50, 40)
			ttV.SetScale(sc.scale)
			ttV.Frame()
			imgV := ttV.Image()
			darkCols := 0
			for x := 0; x < imgV.Bounds().Dx(); x++ {
				if imgV.RGBAAt(x, 20).R < 128 {
					darkCols++
				}
			}
			if darkCols != sc.wantPx {
				t.Fatalf("vertical scale=%g pos=%g: painted %d physical px cols, want %d", sc.scale, pos, darkCols, sc.wantPx)
			}
		}
	}
}

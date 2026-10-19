//go:build windows

package ui

import (
	"image"
	"testing"

	"github.com/egoist/mygo/internal/gpu/d3d11"
)

func TestFixedLineHeightD3DTextBoundsMatchSoftware(t *testing.T) {
	tt := coreNewTester(fixedLineHeightView, 400, 140)
	tt.SetScale(1.5)
	pix, stride, err := d3d11.RenderOffscreen(tt.h.last)
	if err != nil {
		t.Fatal(err)
	}
	software := tt.Image()
	var cpu, gpu image.Rectangle
	for y := range software.Bounds().Dy() {
		for x := range software.Bounds().Dx() {
			p := software.RGBAAt(x, y)
			if p.R < 128 && p.G < 128 && p.B < 128 {
				cpu = cpu.Union(image.Rect(x, y, x+1, y+1))
			}
			i := y*stride + x*4
			if pix[i] < 128 && pix[i+1] < 128 && pix[i+2] < 128 {
				gpu = gpu.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if cpu.Empty() || cpu != gpu {
		t.Fatalf("fixed/mixed text raster bounds: software %v, Direct3D %v", cpu, gpu)
	}
	t.Logf("Direct3D readback and software share text ink bounds %v at 1.5 px/DIP", gpu)
}

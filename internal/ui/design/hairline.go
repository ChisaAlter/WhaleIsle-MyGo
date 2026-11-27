package design

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// PhysicalPixel returns the DIP size of one physical pixel at scale.
func PhysicalPixel(scale float32) float32 {
	if scale <= 0 {
		return 1
	}
	return 1 / scale
}

// HairlinePixels returns the Blink-compatible physical pixel thickness for a
// 1 DIP hairline: 1px at 100%/125%/150%/175% DPI, 2px at 200% DPI.
func HairlinePixels(scale float32) float32 {
	if scale <= 0 {
		return 1
	}
	return max(float32(math.Floor(float64(scale))), 1)
}

// HairlineWidth returns the DIP thickness of a Blink-aligned 1 DIP hairline at
// scale: max(floor(scale), 1) / scale.
func HairlineWidth(scale float32) float32 {
	if scale <= 0 {
		return 1
	}
	return HairlinePixels(scale) / scale
}

// DrawHairline paints a pixel-aligned hairline centered within r without
// changing layout dimensions, preventing 1px/2px odd-even parity jumps at
// fractional DPI scales such as 125%, 150% and 175%.
func DrawHairline(p *ui.Painter, r ui.Rect, vertical bool, col ui.Color) {
	if col.A == 0 || r.W <= 0 || r.H <= 0 {
		return
	}
	scale := p.Scale()
	if scale <= 0 {
		scale = 1
	}
	thickPx := HairlinePixels(scale)
	if vertical {
		x0Px := float32(math.Round(float64((r.X+r.W/2)*scale - thickPx/2)))
		x0 := x0Px / scale
		x1 := (x0Px + thickPx) / scale
		p.Fill(ui.Rect{X: x0, Y: r.Y, W: x1 - x0, H: r.H}, col, 0)
		return
	}
	y0Px := float32(math.Round(float64((r.Y+r.H/2)*scale - thickPx/2)))
	y0 := y0Px / scale
	y1 := (y0Px + thickPx) / scale
	p.Fill(ui.Rect{X: r.X, Y: y0, W: r.W, H: y1 - y0}, col, 0)
}

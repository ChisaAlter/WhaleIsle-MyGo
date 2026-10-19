//go:build windows && (amd64 || arm64)

package windows

import (
	"cmp"
	"math"
)

// Rounded controls use the design-platform light/dark label and hover colors,
// rather than Windows' inactive-caption dimming and version-specific close red.
func (c *captionBar) roundedColors(i int, b captionButton, dark bool) (back, fore rgba) {
	fore = rgba{15, 17, 21, 255}
	back = rgba{0, 0, 0, 1} // retain the entire rectangular native hit target
	if dark {
		fore = rgba{249, 250, 251, 255}
	}
	if !c.w.buttonEnabled(b) {
		fore.a = 0x5C
		return
	}
	if i != c.hot || c.pressed >= 0 && c.pressed != i {
		return
	}
	if b == buttonClose {
		return closeRed10, rgba{255, 255, 255, 255}
	}
	if dark {
		return rgba{255, 255, 255, 20}, fore // rgba(255,255,255,.08)
	}
	return rgba{38, 49, 72, 15}, fore // rgba(38,49,72,.06)
}

// paintRounded rasterizes the compact controls' SVG geometry directly. It does
// not depend on the size or outlines of Segoe Fluent/MDL2 on the host OS. A 4x4
// coverage grid anti-aliases the 12-unit vector glyphs at fractional DPI.
func (c *captionBar) paintRounded(px []uint32, width, height, bw int32, dpi int) {
	o := c.w.opts.WindowsCaptionButtons
	scale := float64(dpi) / 96
	radius := min(float64(max(o.CornerRadius, 0))*scale, min(float64(bw), float64(height))/2)
	glyph := float64(max(cmp.Or(o.GlyphSize, 12), 1)) * scale
	maximized, dark := c.w.IsMaximized(), c.w.b.isDark()
	masks := c.rounded.coverage(bw, height, glyph, radius, maximized)
	for i, b := range c.shown {
		back, fore := c.colors(i, b, dark, false)
		for y := int32(0); y < height; y++ {
			for x := int32(0); x < bw; x++ {
				mask := masks[(int32(b)*height+y)*bw+x]
				plate := back
				plate.a = uint8(max(uint32(back.a)*uint32(mask&255)/255, 1))
				px[y*width+int32(i)*bw+x] = over(fore, uint32(mask>>8), plate)
			}
		}
	}
}

// The SVG coverage is static while geometry and maximize state stay unchanged;
// hover/activation only recolor it, just as the original cached SVG does.
type roundedCaptionRaster struct {
	width, height int32
	glyph, radius float64
	maximized     bool
	mask          []uint16
}

func (r *roundedCaptionRaster) coverage(bw, height int32, glyph, radius float64, maximized bool) []uint16 {
	if r.width == bw && r.height == height && r.glyph == glyph && r.radius == radius && r.maximized == maximized && len(r.mask) > 0 {
		return r.mask
	}
	r.width, r.height, r.glyph, r.radius, r.maximized = bw, height, glyph, radius, maximized
	r.mask = make([]uint16, 3*int(bw)*int(height))
	for b := buttonMinimize; b <= buttonClose; b++ {
		for y := int32(0); y < height; y++ {
			for x := int32(0); x < bw; x++ {
				var bg, fg uint32
				for sy := range 4 {
					for sx := range 4 {
						fx, fy := float64(x)+(float64(sx)+.5)/4, float64(y)+(float64(sy)+.5)/4
						if captionRoundDistance(fx, fy, 0, 0, float64(bw), float64(height), radius) <= 0 {
							bg++
						}
						gx := (fx - (float64(bw)-glyph)/2) * 12 / glyph
						gy := (fy - (float64(height)-glyph)/2) * 12 / glyph
						if roundedCaptionGlyph(b, maximized, gx, gy) {
							fg++
						}
					}
				}
				r.mask[(int32(b)*height+y)*bw+x] = uint16(fg*255/16)<<8 | uint16(bg*255/16)
			}
		}
	}
	return r.mask
}

// captionRoundDistance is the signed distance to an SVG circular rounded rect.
func captionRoundDistance(x, y, left, top, width, height, radius float64) float64 {
	qx := math.Abs(x-left-width/2) - (width/2 - radius)
	qy := math.Abs(y-top-height/2) - (height/2 - radius)
	return math.Hypot(max(qx, 0), max(qy, 0)) + min(max(qx, qy), 0) - radius
}

func roundedCaptionGlyph(b captionButton, maximized bool, x, y float64) bool {
	stroke := func(left, top, width, height, radius, thickness float64) bool {
		return math.Abs(captionRoundDistance(x, y, left, top, width, height, radius)) <= thickness/2
	}
	switch b {
	case buttonMinimize:
		return captionRoundDistance(x, y, 2, 5.4, 8, 1.2, .6) <= 0
	case buttonMaximize:
		if maximized {
			return stroke(3.4, 2.2, 6.2, 6.2, 1.2, 1.15) || stroke(2.2, 3.6, 6.2, 6.2, 1.2, 1.15)
		}
		return stroke(2.4, 2.4, 7.2, 7.2, 1.4, 1.2)
	default:
		return captionSegmentDistance(x, y, 3, 3, 9, 9) <= .625 || captionSegmentDistance(x, y, 9, 3, 3, 9) <= .625
	}
}

func captionSegmentDistance(x, y, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := min(max(((x-ax)*dx+(y-ay)*dy)/(dx*dx+dy*dy), 0), 1)
	return math.Hypot(x-ax-t*dx, y-ay-t*dy)
}

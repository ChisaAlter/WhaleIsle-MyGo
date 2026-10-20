package composer

import (
	"image"
	"math"

	"github.com/egoist/mygo/ui"
)

// projectModelFlip projects one already clipped model-label seat around its
// center, matching motion.css's perspective:140px and rotateX. Both text layers
// use the current label's seat, even when the outgoing label was wider.
//
// src is premultiplied RGBA at scale pixels per DIP; a nonzero Rect.Min is
// supported. The exact logical seat sizes set the transform center, avoiding
// ceil-to-pixel raster padding. The returned offset is in DIPs from the seat's
// top-left, so the widened near edge is not cropped back to the seat.
func projectModelFlip(src *image.RGBA, seatWidth, seatHeight, scale float32, rotateXDeg, opacity float64) (*image.RGBA, ui.Point) {
	if src == nil || src.Bounds().Empty() || seatWidth <= 0 || seatHeight <= 0 || scale <= 0 || opacity <= 0 {
		return nil, ui.Point{}
	}
	sine, cosine := math.Sincos(rotateXDeg * math.Pi / 180)
	const perspective = 140.0
	pixelsPerDIP := float64(scale)
	bounds := src.Bounds()
	width, height := float64(seatWidth), float64(seatHeight)
	centerX, centerY := width/2, height/2
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, corner := range [4][2]float64{{0, 0}, {width, 0}, {0, height}, {width, height}} {
		x, y := corner[0]-centerX, corner[1]-centerY
		denominator := 1 - y*sine/perspective
		projectedX, projectedY := centerX+x/denominator, centerY+y*cosine/denominator
		minX, minY = min(minX, projectedX), min(minY, projectedY)
		maxX, maxY = max(maxX, projectedX), max(maxY, projectedY)
	}
	left, top := int(math.Floor(minX*pixelsPerDIP)), int(math.Floor(minY*pixelsPerDIP))
	right, bottom := int(math.Ceil(maxX*pixelsPerDIP)), int(math.Ceil(maxY*pixelsPerDIP))
	dst := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	sample := func(x, y int) [4]float64 {
		if x < 0 || y < 0 || x >= bounds.Dx() || y >= bounds.Dy() {
			return [4]float64{}
		}
		at := src.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
		return [4]float64{float64(src.Pix[at]), float64(src.Pix[at+1]), float64(src.Pix[at+2]), float64(src.Pix[at+3])}
	}
	for y := range dst.Bounds().Dy() {
		projectedY := (float64(top+y)+.5)/pixelsPerDIP - centerY
		// Invert y'=y*cos(theta)/(1-y*sin(theta)/perspective).
		denominator := cosine + projectedY*sine/perspective
		sourceY := projectedY / denominator
		perspectiveDivisor := 1 - sourceY*sine/perspective
		sampleY := (centerY+sourceY)*pixelsPerDIP - .5
		y0 := int(math.Floor(sampleY))
		fractionY := sampleY - float64(y0)
		for x := range dst.Bounds().Dx() {
			projectedX := (float64(left+x)+.5)/pixelsPerDIP - centerX
			sampleX := (centerX+projectedX*perspectiveDivisor)*pixelsPerDIP - .5
			x0 := int(math.Floor(sampleX))
			fractionX := sampleX - float64(x0)
			a, b := sample(x0, y0), sample(x0+1, y0)
			c, d := sample(x0, y0+1), sample(x0+1, y0+1)
			at := dst.PixOffset(x, y)
			for channel := range 4 {
				upper := a[channel]*(1-fractionX) + b[channel]*fractionX
				lower := c[channel]*(1-fractionX) + d[channel]*fractionX
				// Interpolate premultiplied channels directly, including alpha. Applying
				// opacity to RGB and alpha together avoids dark transparent-edge fringes.
				dst.Pix[at+channel] = uint8(math.Round((upper*(1-fractionY) + lower*fractionY) * opacity))
			}
		}
	}
	return dst, ui.Point{X: float32(float64(left) / pixelsPerDIP), Y: float32(float64(top) / pixelsPerDIP)}
}

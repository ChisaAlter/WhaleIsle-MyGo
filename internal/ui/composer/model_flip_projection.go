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
	dstW, dstH := right-left, bottom-top
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	srcW, srcH := bounds.Dx(), bounds.Dy()
	srcMinX, srcMinY := bounds.Min.X, bounds.Min.Y
	srcPix, srcStride := src.Pix, src.Stride
	dstPix, dstStride := dst.Pix, dst.Stride
	baseOff := src.PixOffset(srcMinX, srcMinY)

	// Identity fast path when flat and opaque.
	if rotateXDeg == 0 && opacity == 1 && left == 0 && top == 0 && dstW == srcW && dstH == srcH {
		for y := 0; y < dstH; y++ {
			so := baseOff + y*srcStride
			do := y * dstStride
			copy(dstPix[do:do+dstW*4], srcPix[so:so+srcW*4])
		}
		return dst, ui.Point{}
	}

	op := float32(opacity)
	invScale := 1.0 / pixelsPerDIP
	for y := 0; y < dstH; y++ {
		projectedY := (float64(top+y)+.5)*invScale - centerY
		// Invert y'=y*cos(theta)/(1-y*sin(theta)/perspective).
		denominator := cosine + projectedY*sine/perspective
		sourceY := projectedY / denominator
		perspectiveDivisor := 1 - sourceY*sine/perspective
		sampleY := (centerY+sourceY)*pixelsPerDIP - .5
		y0 := int(math.Floor(sampleY))
		y1 := y0 + 1
		if y1 < 0 || y0 >= srcH {
			continue
		}
		fy := float32(sampleY - float64(y0))
		fy1 := 1 - fy
		y0In := y0 >= 0 && y0 < srcH
		y1In := y1 >= 0 && y1 < srcH
		row0 := baseOff + y0*srcStride
		row1 := baseOff + y1*srcStride
		dstRow := y * dstStride
		stepX := perspectiveDivisor
		sampleX := (centerX+(float64(left)+.5)*invScale*perspectiveDivisor-centerX*perspectiveDivisor)*pixelsPerDIP - .5
		for x := 0; x < dstW; x++ {
			x0 := int(math.Floor(sampleX))
			x1 := x0 + 1
			fx := float32(sampleX - float64(x0))
			sampleX += stepX
			if x1 < 0 || x0 >= srcW {
				continue
			}
			fx1 := 1 - fx
			x0In := x0 >= 0 && x0 < srcW
			x1In := x1 >= 0 && x1 < srcW

			var a0, a1, a2, a3 float32
			var b0, b1, b2, b3 float32
			var c0, c1, c2, c3 float32
			var d0, d1, d2, d3 float32
			if y0In {
				if x0In {
					o := row0 + x0*4
					a0, a1, a2, a3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
				if x1In {
					o := row0 + x1*4
					b0, b1, b2, b3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
			}
			if y1In {
				if x0In {
					o := row1 + x0*4
					c0, c1, c2, c3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
				if x1In {
					o := row1 + x1*4
					d0, d1, d2, d3 = float32(srcPix[o]), float32(srcPix[o+1]), float32(srcPix[o+2]), float32(srcPix[o+3])
				}
			}
			if a3 == 0 && b3 == 0 && c3 == 0 && d3 == 0 {
				continue
			}
			do := dstRow + x*4
			// Interpolate premultiplied channels directly, including alpha.
			dstPix[do] = uint8(((a0*fx1+b0*fx)*fy1+(c0*fx1+d0*fx)*fy)*op + 0.5)
			dstPix[do+1] = uint8(((a1*fx1+b1*fx)*fy1+(c1*fx1+d1*fx)*fy)*op + 0.5)
			dstPix[do+2] = uint8(((a2*fx1+b2*fx)*fy1+(c2*fx1+d2*fx)*fy)*op + 0.5)
			dstPix[do+3] = uint8(((a3*fx1+b3*fx)*fy1+(c3*fx1+d3*fx)*fy)*op + 0.5)
		}
	}
	return dst, ui.Point{X: float32(float64(left) / pixelsPerDIP), Y: float32(float64(top) / pixelsPerDIP)}
}

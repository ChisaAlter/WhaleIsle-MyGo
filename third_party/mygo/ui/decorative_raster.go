package ui

import "image"

// DecorativeRaster retains one native texture for a paint-owned animated
// decoration. Updates run on the painter's thread; no mutable image is shared
// with a renderer or a background worker.
type DecorativeRaster struct{ bitmap *Bitmap }

func (r *DecorativeRaster) Paint(p *Painter, pixels *image.RGBA, box Rect) {
	if r.bitmap == nil || r.bitmap.w != pixels.Rect.Dx() || r.bitmap.h != pixels.Rect.Dy() {
		r.bitmap = NewBitmap(pixels)
	} else {
		for y := 0; y < pixels.Rect.Dy(); y++ {
			copy(r.bitmap.img.Pix[y*r.bitmap.w*4:], pixels.Pix[y*pixels.Stride:y*pixels.Stride+r.bitmap.w*4])
		}
		r.bitmap.img.Changed()
		r.bitmap.levels = nil
	}
	p.Image(r.bitmap, box, FillBox)
}

package chrome

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The installed official hero uses the very same head as BrandAvatar, with
// mask-mode:alpha and background-color:currentColor. Keep the original alpha
// at source resolution; its blue/white RGB never contributes to the silhouette.
var officialHeroAlpha = sync.OnceValue(func() *image.Alpha {
	src := whaleHeadDecoded()
	if src == nil {
		return nil
	}
	b := src.Bounds()
	alpha := image.NewAlpha(b)
	w, h := b.Dx(), b.Dy()
	switch img := src.(type) {
	case *image.NRGBA:
		for y := 0; y < h; y++ {
			srcRow := img.Pix[y*img.Stride:]
			dstRow := alpha.Pix[y*alpha.Stride : y*alpha.Stride+w]
			for x := 0; x < w; x++ {
				dstRow[x] = srcRow[x*4+3]
			}
		}
	case *image.RGBA:
		for y := 0; y < h; y++ {
			srcRow := img.Pix[y*img.Stride:]
			dstRow := alpha.Pix[y*alpha.Stride : y*alpha.Stride+w]
			for x := 0; x < w; x++ {
				dstRow[x] = srcRow[x*4+3]
			}
		}
	default:
		draw.Draw(alpha, alpha.Bounds(), src, b.Min, draw.Src)
	}
	return alpha
})

var heroBrandCache struct {
	sync.Mutex
	next    int
	entries [16]struct {
		ink    ui.Color
		bitmap *ui.Bitmap
	}
}

// HeroBrandMark returns the official PNG's exact alpha filled with current ink.
// Eight entries cover active primary/muted/light/dark ink without an unbounded theme cache.
// A malformed/missing compiled asset returns nil for the generic hero fallback.
func HeroBrandMark(ink ui.Color) *ui.Bitmap {
	mask := officialHeroAlpha()
	if mask == nil {
		return nil
	}
	heroBrandCache.Lock()
	for _, e := range heroBrandCache.entries {
		if e.bitmap != nil && e.ink == ink {
			bmp := e.bitmap
			heroBrandCache.Unlock()
			return bmp
		}
	}
	heroBrandCache.Unlock()

	bitmap := ui.NewBitmap(heroBrandImage(mask, ink))
	bitmap.WarmLevels(64)

	heroBrandCache.Lock()
	defer heroBrandCache.Unlock()
	for _, e := range heroBrandCache.entries {
		if e.bitmap != nil && e.ink == ink {
			return e.bitmap
		}
	}
	e := &heroBrandCache.entries[heroBrandCache.next]
	e.ink, e.bitmap = ink, bitmap
	heroBrandCache.next = (heroBrandCache.next + 1) % len(heroBrandCache.entries)
	return bitmap
}

func heroBrandImage(mask *image.Alpha, ink ui.Color) *image.RGBA {
	b := mask.Bounds()
	out := image.NewRGBA(b)
	sr, sg, sb, sa := color.NRGBA{R: ink.R, G: ink.G, B: ink.B, A: ink.A}.RGBA()
	var lut [256][4]uint8
	for m := 1; m < 256; m++ {
		ma := uint32(m) * 0x101
		lut[m] = [4]uint8{
			uint8((sr * ma / 0xffff) >> 8),
			uint8((sg * ma / 0xffff) >> 8),
			uint8((sb * ma / 0xffff) >> 8),
			uint8((sa * ma / 0xffff) >> 8),
		}
	}
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		maskRow := mask.Pix[y*mask.Stride : y*mask.Stride+w]
		outRow := out.Pix[y*out.Stride : y*out.Stride+w*4]
		for x, m := range maskRow {
			if m == 0 {
				continue
			}
			c := lut[m]
			j := x * 4
			outRow[j] = c[0]
			outRow[j+1] = c[1]
			outRow[j+2] = c[2]
			outRow[j+3] = c[3]
		}
	}
	return out
}

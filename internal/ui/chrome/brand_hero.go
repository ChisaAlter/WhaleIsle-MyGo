package chrome

import (
	"image"
	"image/color"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The installed official hero uses the very same head as BrandAvatar, with
// mask-mode:alpha and background-color:currentColor. Keep the 1-byte source
// alpha silhouette (1.5 MiB) so HeroBrandMark can stream-tint directly into
// compact 157×157 mip-level bitmaps (98 KiB each) with bit-exact filtering and
// zero full-resolution RGBA allocations.
func officialHeroAlpha() *image.Alpha {
	loadWhaleBrandAssets()
	return whaleHeroAlpha
}

var heroBrandCache struct {
	sync.Mutex
	next    int
	entries [16]struct {
		ink    ui.Color
		bitmap *ui.Bitmap
	}
}

func resetHeroBrandCacheForTest() {
	heroBrandCache.Lock()
	heroBrandCache.entries = [16]struct {
		ink    ui.Color
		bitmap *ui.Bitmap
	}{}
	heroBrandCache.next = 0
	heroBrandCache.Unlock()
}

// HeroBrandMark returns the official PNG's alpha silhouette filled with current
// ink, stream-downsampled lazily to its compact 157×157 display mip level (98
// KiB) and cached per ink. A malformed/missing compiled asset returns nil for
// the generic hero fallback.
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

	bitmap := ui.NewBitmap(heroBrandMipImage(mask, ink))
	bitmap.WarmLevels(8)

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

func heroBrandLUT(ink ui.Color) [256][4]uint8 {
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
	return lut
}

// heroBrandMipImage stream-tints mask directly into its 3× halved mip level
// (157×157 for the 1254×1254 mascot) in stack tiles, producing pixels bit-exact
// to premultiplying at full resolution then halving while allocating only 98 KiB.
func heroBrandMipImage(mask *image.Alpha, ink ui.Color) *image.RGBA {
	b := mask.Bounds()
	w, h := b.Dx(), b.Dy()
	lut := heroBrandLUT(ink)
	pix, stride := mask.Pix, mask.Stride
	return halve3Tiled(w, h, func(x, y int) [4]uint8 {
		return lut[pix[y*stride+x]]
	})
}

func heroBrandImage(mask *image.Alpha, ink ui.Color) *image.RGBA {
	b := mask.Bounds()
	out := image.NewRGBA(b)
	lut := heroBrandLUT(ink)
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

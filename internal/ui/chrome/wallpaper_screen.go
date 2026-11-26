package chrome

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	xdraw "golang.org/x/image/draw"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/wallpapers"
)

// wallpaperMaxPixels caps the decoded wallpaper texture: the window never
// shows more than a 4K screen's worth of it.
const wallpaperMaxPixels = 3840

// wallpaperBucket rounds a window size for the pixelated variant, so a live
// resize does not decode a new texture for every pixel of drag.
const wallpaperBucket = 64

// colorMapColumns is the color map's width in cells; rows follow the image's
// aspect. Surfaces sample this map, never the wallpaper's pixels.
const colorMapColumns = 48

// wallpaperColors is the small precomputed color map of one prepared
// wallpaper: the mean color of each cell, made once on a media worker when the
// wallpaper (or its pixelation) changes.
type wallpaperColors struct {
	cols, rows int
	rgb        []uint8
}

// wallpaperMedia is the process image service; tests substitute their own.
var wallpaperMedia = media.Default

// wallpaperRequest asks the media service for the wallpaper as the window
// paints it: decoded once (capped at 4K), pixelated on the worker when the
// pixelate slider asks for it, with its color map. Blur is not baked in: the
// painter applies it on the GPU.
func wallpaperRequest(path string, p wallpapers.Preferences, w, h int) *media.Handle {
	var key media.Key
	var src media.Source
	if strings.HasPrefix(path, "data:") {
		key, src = media.DataURL(path)
	} else {
		key, src = media.File(path, "")
	}
	src.MaxBytes = 64 << 20
	block := int(math.Round(wallpaperPixelFactor(p.Pixelate)))
	if wallpaperKind(p) != "wallpaper" && w > 0 && h > 0 {
		bw := max(wallpaperBucket, (w+wallpaperBucket-1)/wallpaperBucket*wallpaperBucket)
		bh := max(wallpaperBucket, (h+wallpaperBucket-1)/wallpaperBucket*wallpaperBucket)
		key = key.Sized(bw, bh)
		key.Appearance = wallpaperKind(p)
		src.Derive = func(img image.Image) (image.Image, any) {
			out := pixelateCover(img, bw, bh, block)
			return out, colorMap(out)
		}
	} else {
		key.Appearance = "wallpaper"
		src.Derive = func(img image.Image) (image.Image, any) {
			out := capImage(img, wallpaperMaxPixels)
			return out, colorMap(out)
		}
	}
	svc := wallpaperMedia()
	handle := svc.Request(nil, key, src)
	if wallpaperSettle != nil {
		wallpaperSettle(svc)
	}
	return handle
}

// wallpaperSettle, set only by tests, runs queued decodes before painting.
var wallpaperSettle func(*media.Service)

// wallpaperKind names the prepared variant the preferences ask for.
func wallpaperKind(p wallpapers.Preferences) string {
	if block := int(math.Round(wallpaperPixelFactor(p.Pixelate))); block > 1 {
		return fmt.Sprintf("wallpaper/pixelate:%d", block)
	}
	return "wallpaper"
}

// capImage scales img down so neither side exceeds limit.
func capImage(img image.Image, limit int) image.Image {
	b := img.Bounds()
	scale := math.Min(1, math.Min(float64(limit)/float64(b.Dx()), float64(limit)/float64(b.Dy())))
	if scale >= 1 {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))))
	xdraw.ApproxBiLinear.Scale(dst, dst.Rect, img, b, draw.Src, nil)
	return dst
}

// pixelateCover is the page's pixelate on screen pixels: cover-fit into w×h
// at 1/block of the size, then a nearest-neighbour upscale, so the block size
// does not depend on the source image's size.
func pixelateCover(img image.Image, w, h, block int) image.Image {
	b := img.Bounds()
	sw, sh := max(1, (w+block-1)/block), max(1, (h+block-1)/block)
	scale := math.Max(float64(sw)/float64(b.Dx()), float64(sh)/float64(b.Dy()))
	cw, ch := float64(sw)/scale, float64(sh)/scale
	crop := image.Rect(int(float64(b.Min.X)+(float64(b.Dx())-cw)/2), int(float64(b.Min.Y)+(float64(b.Dy())-ch)/2), 0, 0)
	crop.Max = image.Point{X: crop.Min.X + int(math.Max(1, cw)), Y: crop.Min.Y + int(math.Max(1, ch))}
	small := image.NewRGBA(image.Rect(0, 0, sw, sh))
	xdraw.ApproxBiLinear.Scale(small, small.Rect, img, crop.Intersect(b), draw.Src, nil)
	out := image.NewRGBA(image.Rect(0, 0, sw*block, sh*block))
	xdraw.NearestNeighbor.Scale(out, out.Rect, small, small.Rect, draw.Src, nil)
	return out
}

// colorMap averages img into colorMapColumns columns and as many rows as its
// aspect gives, striding large cells so the pass stays cheap.
func colorMap(img image.Image) *wallpaperColors {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil
	}
	cols := min(colorMapColumns, b.Dx())
	rows := max(1, min(b.Dy(), int(math.Round(float64(cols)*float64(b.Dy())/float64(b.Dx())))))
	m := &wallpaperColors{cols: cols, rows: rows, rgb: make([]uint8, 3*cols*rows)}
	rgba, _ := img.(*image.RGBA)
	for y := 0; y < rows; y++ {
		y0, y1 := b.Min.Y+y*b.Dy()/rows, b.Min.Y+(y+1)*b.Dy()/rows
		for x := 0; x < cols; x++ {
			x0, x1 := b.Min.X+x*b.Dx()/cols, b.Min.X+(x+1)*b.Dx()/cols
			step := max(1, int(math.Sqrt(float64((x1-x0)*(y1-y0))/64)))
			var r, g, bl, n uint64
			for sy := y0; sy < max(y1, y0+1); sy += step {
				for sx := x0; sx < max(x1, x0+1); sx += step {
					if rgba != nil {
						i := rgba.PixOffset(sx, sy)
						r, g, bl = r+uint64(rgba.Pix[i]), g+uint64(rgba.Pix[i+1]), bl+uint64(rgba.Pix[i+2])
					} else {
						cr, cg, cb, _ := img.At(sx, sy).RGBA()
						r, g, bl = r+uint64(cr>>8), g+uint64(cg>>8), bl+uint64(cb>>8)
					}
					n++
				}
			}
			i := 3 * (y*cols + x)
			m.rgb[i], m.rgb[i+1], m.rgb[i+2] = uint8(r/n), uint8(g/n), uint8(bl/n)
		}
	}
	return m
}

// average returns the mean color of the cells under panel, for a wallpaper
// drawn with ui.Cover in box, scaled by keep (the share the dim mask leaves).
func (m *wallpaperColors) average(box, panel ui.Rect, keep float32) (ui.Color, bool) {
	if m == nil {
		return ui.Color{}, false
	}
	region, ok := coverRegion(box, panel, image.Rect(0, 0, m.cols, m.rows))
	if !ok {
		return ui.Color{}, false
	}
	var r, g, b, n float32
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			i := 3 * (y*m.cols + x)
			r, g, b, n = r+float32(m.rgb[i]), g+float32(m.rgb[i+1]), b+float32(m.rgb[i+2]), n+1
		}
	}
	if n == 0 {
		return ui.Color{}, true
	}
	mix := func(v float32) uint8 {
		return uint8(math.Max(0, math.Min(255, math.Round(float64(v/n*keep)))))
	}
	return ui.RGB(mix(r), mix(g), mix(b)), true
}

// wallpaperBlurMaterial is the wallpaper blur slider on the GPU: the page's
// filter: blur(radius) over the painted image.
func wallpaperBlurMaterial(radius int) glass.Blur {
	sat := float32(0)
	if radius > 0 {
		sat = min(float32(0.35), float32(radius)/40*0.35)
	}
	return glass.Blur{Radius: float32(radius) / 2, Saturation: sat, Dither: true}
}

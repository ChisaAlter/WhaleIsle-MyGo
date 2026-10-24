package chrome

import (
	"image"
	"image/color"
	"math"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
)

// screenBleed matches WALLPAPER_BLEED in the installed wallpaper layer: the
// sheet paints the bitmap 48px outside the window on every side so a blur has
// real neighbours instead of the clamped edge, then clips it back.
const screenBleed = 48

// backgroundScreenBitmap prepares the wallpaper at the window's own pixel size:
// cover-fit into a 48px-bled canvas, pixelate by the screen block size, blur,
// then crop the bleed back. Pixelation therefore happens after cover-fit, as
// the page does (it draws the bitmap at cssSize/factor and stretches it back),
// so the on-screen block size no longer depends on the source image size.
func backgroundScreenBitmap(path string, p wallpapers.Preferences, w, h int) *ui.Bitmap {
	s := &backgroundState
	if w <= 0 || h <= 0 {
		return nil
	}
	if s.path == path && s.targetW == w && s.targetH == h && s.blur == p.Blur && s.pixel == p.Pixelate && s.bitmap != nil {
		return s.bitmap
	}
	src := decodedWallpaper(path)
	if src == nil {
		s.path, s.targetW, s.targetH, s.pixels, s.bitmap = path, w, h, nil, nil
		s.version++
		return nil
	}
	fullW, fullH := w+2*screenBleed, h+2*screenBleed
	dst := coverFit(src, fullW, fullH)
	if block := int(math.Round(wallpaperPixelFactor(p.Pixelate))); block > 1 {
		dst = pixelateBlocks(dst, block)
	}
	if radius := wallpaperBlurRadius(p.Blur); radius > 0 {
		dst = blurThreePass(dst, radius)
	}
	if screenBleed > 0 && (fullW != w || fullH != h) {
		dst = cropImage(dst, image.Rect(screenBleed, screenBleed, screenBleed+w, screenBleed+h))
	}
	s.path, s.targetW, s.targetH = path, w, h
	s.blur, s.pixel = p.Blur, p.Pixelate
	s.bitmap = ui.NewBitmap(dst)
	s.version++
	return s.bitmap
}

// decodedWallpaper keeps the decoded source so resizing the window does not
// re-read and re-decode the file.
func decodedWallpaper(path string) *image.RGBA {
	s := &backgroundState
	if s.decodedPath == path && s.decoded != nil {
		return s.decoded
	}
	s.decodedPath, s.decoded = path, nil
	data, err := wallpaperImageBytes(path)
	if err != nil {
		return nil
	}
	img, err := wallpapers.Decode(data)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.SetRGBA(x, y, img.At(b.Min.X+x, b.Min.Y+y).(color.RGBA))
		}
	}
	s.decoded = dst
	return dst
}

// coverFit scales src to cover w×h and centre-crops it, sampling bilinearly the
// way the page's canvas smoothing does before its pixelated upscale.
func coverFit(src *image.RGBA, w, h int) *image.RGBA {
	sb := src.Bounds()
	scale := math.Max(float64(w)/float64(sb.Dx()), float64(h)/float64(sb.Dy()))
	sw, sh := float64(sb.Dx())*scale, float64(sb.Dy())*scale
	offX, offY := (sw-float64(w))/2, (sh-float64(h))/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx := (float64(x)+.5+offX)/scale - .5
			fy := (float64(y)+.5+offY)/scale - .5
			dst.SetRGBA(x, y, bilinear(src, fx, fy))
		}
	}
	return dst
}

func bilinear(src *image.RGBA, fx, fy float64) color.RGBA {
	b := src.Bounds()
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	at := func(x, y int) color.RGBA {
		return src.RGBAAt(min(max(x, 0), b.Dx()-1), min(max(y, 0), b.Dy()-1))
	}
	c00, c10, c01, c11 := at(x0, y0), at(x0+1, y0), at(x0, y0+1), at(x0+1, y0+1)
	mix := func(a, b, t float64) float64 { return a + (b-a)*t }
	blend := func(a, b, c, d uint8) uint8 {
		v := mix(mix(float64(a), float64(b), tx), mix(float64(c), float64(d), tx), ty) + .5
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v)
	}
	return color.RGBA{R: blend(c00.R, c10.R, c01.R, c11.R), G: blend(c00.G, c10.G, c01.G, c11.G), B: blend(c00.B, c10.B, c01.B, c11.B), A: 255}
}

// pixelateBlocks is the page's downsample-then-nearest-upscale on screen pixels.
func pixelateBlocks(src *image.RGBA, block int) *image.RGBA {
	if block <= 1 {
		return src
	}
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := 0; y < b.Dy(); y += block {
		for x := 0; x < b.Dx(); x += block {
			col := src.RGBAAt(min(x+block/2, b.Dx()-1), min(y+block/2, b.Dy()-1))
			for yy := y; yy < min(y+block, b.Dy()); yy++ {
				for xx := x; xx < min(x+block, b.Dx()); xx++ {
					dst.SetRGBA(xx, yy, col)
				}
			}
		}
	}
	return dst
}

// blurThreePass approximates the page's CSS gaussian blur(radius) with three
// box passes: three boxes of half-width b add up to a gaussian of variance
// 3*((2b+1)^2-1)/12, matched to the sheet's sigma = radius/2.
func blurThreePass(src *image.RGBA, radius int) *image.RGBA {
	if radius <= 0 {
		return src
	}
	sigma := float64(radius) / 2
	box := int(math.Round((math.Sqrt(1+4*sigma*sigma) - 1) / 2))
	if box < 1 {
		box = 1
	}
	out := src
	for pass := 0; pass < 3; pass++ {
		out = blurBitmap(out, box)
	}
	return out
}

func cropImage(src *image.RGBA, r image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			dst.SetRGBA(x, y, src.RGBAAt(r.Min.X+x, r.Min.Y+y))
		}
	}
	return dst
}

// backgroundSamplingBitmap prepares the source-resolution wallpaper the glass
// surfaces average (their sampler maps the window box onto these pixels). The
// painted layer is screen-resolution; sampling stays source-resolution so panel
// materials keep the same numbers they had before the screen-space pass.
func backgroundSamplingBitmap(path string, p wallpapers.Preferences) {
	s := &backgroundState
	if s.sampPath == path && s.sampBlur == p.Blur && s.sampPixel == p.Pixelate && s.sampBmp != nil {
		return
	}
	src := decodedWallpaper(path)
	if src == nil {
		s.sampPath, s.sampBmp, s.pixels = path, nil, nil
		return
	}
	dst := src
	if block := int(math.Round(wallpaperPixelFactor(p.Pixelate))); block > 1 {
		dst = pixelateBlocks(dst, block)
	}
	if radius := wallpaperBlurRadius(p.Blur); radius > 0 {
		dst = blurBitmap(dst, radius)
	}
	s.sampPath, s.sampBlur, s.sampPixel = path, p.Blur, p.Pixelate
	s.sampBmp = ui.NewBitmap(dst)
	s.pixels = dst
}

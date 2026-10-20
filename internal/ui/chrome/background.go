package chrome

import (
	"github.com/egoist/mygo/ui"
	"image"
	"image/draw"
	"math"
	"whaleisle/internal/desktop/wallpapers"
)

var backgroundState struct {
	prefs       wallpapers.Preferences
	active      bool
	wallpaper   string
	path        string
	blur, pixel float64
	bitmap      *ui.Bitmap
	// pixels is the prepared wallpaper the surfaces sample, the box it was
	// drawn in, and the scheme that decided its dim mask. version changes
	// whenever a surface's frozen patch would go stale.
	pixels  *image.RGBA
	box     ui.Rect
	hasBox  bool
	dark    bool
	version uint64
}

func ConfigureBackdrop(path string, p wallpapers.Preferences) {
	backgroundState.prefs = p
	backgroundState.wallpaper = path
	backgroundState.active = path != "" || p.Effect == "gradient"
	if !backgroundState.active {
		backgroundState.pixels, backgroundState.hasBox = nil, false
		backgroundState.version++
	}
}

// effectivePrefs applies the transparent theme only while a wallpaper is live,
// as the harness does (html[data-dsh-transparent] is effective with a wallpaper
// and ignored over the ambient gradient), so the chrome is never cleared over a
// backdrop that has nothing to show through it.
func effectivePrefs(p wallpapers.Preferences) wallpapers.Preferences {
	if p.Transparent && backgroundState.wallpaper == "" {
		p.Transparent = false
	}
	return p
}

// The legacy fixed-alpha surface helpers keep the solidity the wallpaper tokens
// give each layer; MaterialAlpha owns the curves and Glass paints the sampled
// version of the same material.
func BackdropCanvas(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialCanvas, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}
func BackdropSidebar(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialSidebar, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}
func BackdropSurface(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialPanel, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}
func BackdropTerminal(c ui.Color) ui.Color {
	if !backgroundState.active {
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialTerminal, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}

// The caller captures the desktop preference during view construction.
// wallpaper.css leaves the ambient blooms static under reduced motion.
//
// The wallpaper layer paints in the sheet's order: the cover-cropped bitmap,
// then the dim mask (--dsw-alias-bg-mask-1) that keeps text readable over it.
// The transparent theme drops the mask, so the image's colors show through the
// see-through chrome. Surfaces painted after it sample this same layer (see
// glass.go), so the mask is part of what they sample.
func WallpaperBackgroundPainter(path string, p wallpapers.Preferences, reducedMotion ...bool) func(*ui.Painter, ui.Rect) {
	ConfigureBackdrop(path, p)
	reduced := len(reducedMotion) > 0 && reducedMotion[0]
	if path != "" {
		bmp := backgroundBitmap(path, p)
		return func(painter *ui.Painter, r ui.Rect) {
			if bmp == nil {
				return
			}
			painter.Image(bmp, r, ui.Cover)
			s := &backgroundState
			s.box, s.hasBox, s.dark = r, true, painter.Theme().Dark
			if p.Transparent {
				return
			}
			painter.Fill(r, ui.RGBA(0, 0, 0, MaskAlpha(s.dark)), 0)
		}
	}
	if p.Effect != "gradient" {
		return func(*ui.Painter, ui.Rect) {}
	}
	return func(painter *ui.Painter, r ui.Rect) {
		colors := p.Colors
		resolved := [7]ui.Color{painter.Theme().Background, painter.Theme().Surface, painter.Theme().Accent, ui.Hex("#a78bfa"), ui.Hex("#38bdf8"), ui.Hex("#f472b6"), ui.Hex("#34d399")}
		for i := range colors {
			if colors[i] != "" {
				resolved[i] = Color(colors[i])
			}
		}
		t := float64(0)
		if !reduced {
			t = float64(painter.Now().UnixMilli()) / 1000 * p.Speed / 100
		}
		painter.FillGradient(r, ui.LinearGradient{From: resolved[0], To: resolved[1], Angle: 40 + float32(math.Sin(t*.05)*20)}, 0)
		for i := 0; i < min(5, max(1, int(p.Count))); i++ {
			phase := float64(i) * 1.73
			x := r.X + r.W*float32(.5+.36*math.Sin(t*.13+phase))
			y := r.Y + r.H*float32(.5+.35*math.Cos(t*.11+phase))
			w, h := r.W*.45, r.H*.55
			radius := min(w, h) / 2
			switch p.Variant {
			case "aurora":
				w = r.W * .8
				h = r.H * .18
				radius = h / 2
			case "rays":
				w = r.W * .18
				h = r.H * 1.4
				radius = 0
			case "chaos":
				w *= float32(.7 + .3*math.Sin(t*.2+phase))
				h *= float32(.8 + .4*math.Cos(t*.17+phase))
			}
			painter.Shadow(ui.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}, radius, 0, 0, 60, 0, resolved[i+2].Alpha(.38))
		}
		if !reduced {
			painter.AnimationFrame()
		}
	}
}
func backgroundBitmap(path string, p wallpapers.Preferences) *ui.Bitmap {
	s := &backgroundState
	if s.path == path && s.blur == p.Blur && s.pixel == p.Pixelate && s.bitmap != nil {
		return s.bitmap
	}
	data, err := wallpaperImageBytes(path)
	if err != nil {
		s.path, s.pixels, s.bitmap = path, nil, nil
		s.version++
		return nil
	}
	img, err := wallpapers.Decode(data)
	if err != nil {
		s.path, s.pixels, s.bitmap = path, nil, nil
		s.version++
		return nil
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	block := 1 + int(p.Pixelate/100*19)
	if block > 1 {
		for y := 0; y < b.Dy(); y += block {
			for x := 0; x < b.Dx(); x += block {
				col := dst.RGBAAt(min(x+block/2, b.Dx()-1), min(y+block/2, b.Dy()-1))
				for yy := y; yy < min(y+block, b.Dy()); yy++ {
					for xx := x; xx < min(x+block, b.Dx()); xx++ {
						dst.SetRGBA(xx, yy, col)
					}
				}
			}
		}
	}
	radius := int(p.Blur * .4)
	if radius > 0 {
		dst = blurBitmap(dst, radius)
	}
	s.path, s.blur, s.pixel = path, p.Blur, p.Pixelate
	s.bitmap = ui.NewBitmap(dst)
	s.pixels = dst
	s.version++
	return s.bitmap
}

// Separable running-sum blur keeps slider updates linear in the image size.
func blurBitmap(src *image.RGBA, radius int) *image.RGBA {
	b := src.Bounds()
	tmp := image.NewRGBA(b)
	dst := image.NewRGBA(b)
	for pass := 0; pass < 2; pass++ {
		in, out := src, tmp
		length, lines := b.Dx(), b.Dy()
		if pass == 1 {
			in, out = tmp, dst
			length, lines = b.Dy(), b.Dx()
		}
		for line := 0; line < lines; line++ {
			var sum [4]int
			at := func(pos int) int {
				if pass == 0 {
					return in.PixOffset(pos, line)
				}
				return in.PixOffset(line, pos)
			}
			for pos := 0; pos <= min(radius, length-1); pos++ {
				idx := at(pos)
				for ch := 0; ch < 4; ch++ {
					sum[ch] += int(in.Pix[idx+ch])
				}
			}
			for pos := 0; pos < length; pos++ {
				n := min(length-1, pos+radius) - max(0, pos-radius) + 1
				idx := at(pos)
				for ch := 0; ch < 4; ch++ {
					out.Pix[idx+ch] = byte(sum[ch] / n)
				}
				drop, add := pos-radius, pos+radius+1
				if drop >= 0 {
					idx = at(drop)
					for ch := 0; ch < 4; ch++ {
						sum[ch] -= int(in.Pix[idx+ch])
					}
				}
				if add < length {
					idx = at(add)
					for ch := 0; ch < 4; ch++ {
						sum[ch] += int(in.Pix[idx+ch])
					}
				}
			}
		}
	}
	return dst
}

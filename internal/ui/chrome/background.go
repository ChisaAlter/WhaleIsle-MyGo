package chrome

import (
	"github.com/egoist/mygo/ui"
	"image"
	"image/color"
	"math"
	"sync"
	"whaleisle/internal/desktop/wallpapers"
)

var backgroundState struct {
	prefs       wallpapers.Preferences
	active      bool
	wallpaper   string
	path        string
	blur, pixel float64
	bitmap      *ui.Bitmap
	// The bitmap is prepared at the window's own pixel size, so its cache key
	// carries that size alongside the source and the two effect sliders.
	targetW, targetH int
	decodedPath      string
	decoded          *image.RGBA
	// samp* caches the source-resolution bitmap the glass surfaces sample; the
	// painted bitmap above is screen-resolution.
	sampPath  string
	sampBlur  float64
	sampPixel float64
	sampBmp   *ui.Bitmap
	// pixels is the prepared wallpaper the surfaces sample, the box it was
	// drawn in, and the scheme that decided its dim mask. version changes
	// whenever a surface's frozen patch would go stale.
	pixels  *image.RGBA
	box     ui.Rect
	hasBox  bool
	dark    bool
	version uint64
	// transparent is whether the transparent theme was effective on the last
	// configure, so the sharp-wallpaper lift runs once per transition; liftedBlur
	// is what it raised the blur to while it stays effective.
	transparent bool
	liftedBlur  float64
}

// transparentMinBlur is TRANSPARENT_MIN_BLUR (ui-theme/src/wallpaper.ts): the
// blur the harness lifts the wallpaper to when the transparent theme becomes
// effective, so text is not left on a sharp image with 0% surfaces.
const transparentMinBlur = 20

// clampWallpaperEffect matches the harness clampWallpaperEffect(): non-finite
// input becomes 0, everything else is rounded and clamped to the 0–100 slider.
func clampWallpaperEffect(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Min(100, math.Max(0, math.Round(value)))
}

// wallpaperBlurRadius maps the blur slider the way ui-theme does: the page sets
// filter: blur(percent/100*40), so the shell blurs by the same radius.
func wallpaperBlurRadius(percent float64) int {
	return int(math.Round(clampWallpaperEffect(percent) / 100 * 40))
}

// wallpaperPixelFactor maps the pixelate slider to the harness's scale factor:
// 1 at 0, up to 20 at 100 (the page draws the bitmap at size/factor and stretches
// it back with image-rendering: pixelated).
func wallpaperPixelFactor(percent float64) float64 {
	clamped := clampWallpaperEffect(percent)
	if clamped <= 0 {
		return 1
	}
	return 1 + clamped/100*19
}

func ConfigureBackdrop(path string, p wallpapers.Preferences) {
	p = wallpapers.NormalizeEffect(path, p)
	// The harness raises a too-sharp wallpaper once, when the transparent theme
	// becomes effective, and the raise holds while it stays effective: the saved
	// preferences keep the user's own blur, so this only changes what the window
	// paints, and turning the theme off restores it.
	p.Blur = clampWallpaperEffect(p.Blur)
	p.Pixelate = clampWallpaperEffect(p.Pixelate)
	transparent := p.Transparent && path != ""
	if transparent && !backgroundState.transparent && p.Blur < transparentMinBlur {
		backgroundState.liftedBlur = transparentMinBlur
	}
	if !transparent {
		backgroundState.liftedBlur = 0
	}
	if transparent && backgroundState.liftedBlur > p.Blur {
		p.Blur = backgroundState.liftedBlur
	}
	backgroundState.transparent = transparent
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
	// ConfigureBackdrop owns the effective preferences: the transparent theme
	// may have lifted the blur (transparentMinBlur), and the bitmap has to be
	// built with that value.
	p = backgroundState.prefs
	reduced := len(reducedMotion) > 0 && reducedMotion[0]
	if path != "" {
		return func(painter *ui.Painter, r ui.Rect) {
			w, h := int(math.Round(float64(r.W))), int(math.Round(float64(r.H)))
			backgroundSamplingBitmap(path, p)
			bmp := backgroundScreenBitmap(path, p, w, h)
			if bmp == nil {
				return
			}
			// The bitmap is already the window's pixel size: FillBox keeps the
			// screen-space pixel blocks and blur radius exact.
			painter.Image(bmp, r, ui.Cover)
			s := &backgroundState
			s.box, s.hasBox, s.dark = r, true, painter.Theme().Dark
			if p.Transparent {
				return
			}
			painter.Fill(r, ui.RGBA(0, 0, 0, MaskAlpha(s.dark)), 0)
		}
	}
	// Strict gate: a preset or colours alone must never paint; only an explicit
	// gradient does (NormalizeEffect has already resolved unset -> none).
	if p.Effect != "gradient" {
		return func(*ui.Painter, ui.Rect) {}
	}
	return func(painter *ui.Painter, r ui.Rect) {
		colors := p.Colors
		if p.Preset == "dune" {
			// Dune is a fixed product palette; theme colors must not leak the
			// accent blue into its grey-green wash.
			colors = wallpapers.PresetColors("dune")
		} else if colors == [7]string{} && p.Preset != "" {
			colors = wallpapers.PresetColors(p.Preset)
		}
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
		// The wash runs top to bottom, the axis the sheet's gradient band fills,
		// and the blooms sit where the installed build's backdrop has its colour:
		// the bright centre, the greens of the lower left and floor, the grey-blue
		// right flank and the cooler top. They drift a tenth of the window, so the
		// composition stays the sampled one while it still flows.
		painter.FillGradient(r, ui.LinearGradient{From: resolved[0], To: resolved[1], Angle: 180}, 0)
		anchors := [5][2]float32{{.5, .34}, {.2, .82}, {.5, .96}, {.86, .56}, {.72, .1}}
		for i := 0; i < min(5, max(1, int(p.Count))); i++ {
			phase := float64(i) * 1.73
			anchor := anchors[i]
			x := r.X + r.W*(anchor[0]+float32(.1*math.Sin(t*.22+phase)))
			y := r.Y + r.H*(anchor[1]+float32(.08*math.Cos(t*.19+phase)))
			w, h := r.W*.78, r.H*.78
			if p.Variant == "chaos" {
				w *= float32(.8 + .2*math.Sin(t*.13+phase))
				h *= float32(.85 + .2*math.Cos(t*.11+phase))
			}
			// A soft radial bloom, not a blurred shadow: a shadow's cast box
			// leaves the hard circle edge the first pass saw over the wallpaper.
			painter.Image(bloomBitmap(resolved[i+2]), ui.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}, ui.FillBox)
		}
		if !reduced {
			painter.AnimationFrame()
		}
		// The installed sheet dims its gradient with bg-mask-1 so the pale
		// dune wash stays behind the chrome instead of washing the window out.
		if !painter.Theme().Dark {
			painter.Fill(r, ui.RGBA(0, 0, 0, .22), 0)
		}
	}
}

// bloomCache keeps the radial glows of the ambient gradient, one per colour.
var bloomCache sync.Map // ui.Color -> *ui.Bitmap

// bloomDensity scales every bloom's own alpha. The pass-6 comparison against
// evidence/compare/ref-hero-1440.png put the first pass about a tenth too dense:
// the installed backdrop reads milkier at the top left and paler at the
// blue-violet lower right, so each bloom lost the same 12% (255 -> 224 of its
// alpha). Hue, position, count and radius are unchanged.
const bloomDensity = 224

// bloomBitmap is a soft radial glow of c: its colour at the centre fading to
// nothing at the rim along a metaball falloff, so the gradient behind it never
// shows an edge where the bloom ends. Colours without an alpha of their own
// take an opaque centre, scaled by bloomDensity like every other bloom.
func bloomBitmap(c ui.Color) *ui.Bitmap {
	if bmp, ok := bloomCache.Load(c); ok {
		return bmp.(*ui.Bitmap)
	}
	const n = 96
	alpha := c.A
	if alpha == 0 {
		alpha = 255
	}
	alpha = uint8(int(alpha) * bloomDensity / 255)
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx := (float64(x)+.5)/n*2 - 1
			dy := (float64(y)+.5)/n*2 - 1
			d2 := dx*dx + dy*dy
			if d2 >= 1 {
				continue
			}
			// (1 - d²)³ reaches zero with a zero slope at the rim, which is what
			// keeps the bloom's edge out of the picture. The pixels are
			// premultiplied, as every surface bitmap is.
			falloff := 1 - d2
			falloff = falloff * falloff * falloff
			a := uint8(float64(alpha) * falloff)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(int(c.R) * int(a) / 255),
				G: uint8(int(c.G) * int(a) / 255),
				B: uint8(int(c.B) * int(a) / 255),
				A: a,
			})
		}
	}
	bmp := ui.NewBitmap(img)
	bloomCache.Store(c, bmp)
	return bmp
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

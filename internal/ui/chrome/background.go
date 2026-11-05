package chrome

import (
	"github.com/egoist/mygo/ui"
	"image"
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
	screenRGBA  *image.RGBA
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
	// frost* caches the window-space Gaussian-blurred + saturate(150%) bitmap
	// that MaterialMenu (backdrop-filter: blur(40px) saturate(150%)) paints.
	frostSrc         *image.RGBA
	frostBmp         *ui.Bitmap
	frostDark        bool
	frostTransparent bool
	frostVersion     uint64
	// pixels is the prepared wallpaper the surfaces sample, the box it was
	// drawn in, and the scheme that decided its dim mask. version changes
	// whenever a surface's frozen patch would go stale.
	pixels  *image.RGBA
	box     ui.Rect
	hasBox  bool
	dark    bool
	version uint64
}

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
	gradient := path == "" && p.Effect == "gradient"
	if !gradient {
		clearGradientBackdrop()
		ambientGradientFrames.reset()
	}
	// The preference writer performs the one-shot readability nudge when the
	// transparent theme becomes effective. Rendering must honor later explicit
	// blur edits, including values below that initial nudge.
	p.Blur = clampWallpaperEffect(p.Blur)
	p.Pixelate = clampWallpaperEffect(p.Pixelate)
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
		prefs := effectivePrefs(backgroundState.prefs)
		if alpha := ShowThroughAlpha(prefs, backgroundState.dark); alpha < 1 {
			return c.Alpha(alpha)
		}
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialCanvas, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}
func BackdropSidebar(c ui.Color) ui.Color {
	if !backgroundState.active {
		prefs := effectivePrefs(backgroundState.prefs)
		if alpha := ShowThroughAlpha(prefs, backgroundState.dark); alpha < 1 {
			return c.Alpha(alpha)
		}
		return c
	}
	return c.Alpha(MaterialAlpha(MaterialSidebar, effectivePrefs(backgroundState.prefs), backgroundState.dark))
}
func BackdropSurface(c ui.Color) ui.Color {
	if !backgroundState.active {
		prefs := effectivePrefs(backgroundState.prefs)
		if alpha := ShowThroughAlpha(prefs, backgroundState.dark); alpha < 1 {
			return c.Alpha(alpha)
		}
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
	// ConfigureBackdrop owns the effective clamped wallpaper preferences.
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
		config := gradientConfiguration(p, float64(r.W), float64(r.H), painter.Theme().Dark, reduced)
		// latest never renders a phase advance on this thread (6-12 ms of CPU, most
		// of a 60 Hz frame); the finished frame is picked up by the next paint.
		if frame := ambientGradientFrames.latest(config, painter.Now()); frame != nil {
			painter.Image(frame.bitmap, r, ui.FillBox)
			publishGradientBackdrop(frame, r)
		}
		if !reduced {
			// The phase only changes every gradientFrameInterval, so repainting at
			// the display rate (AnimationFrame) painted the same frame twice for
			// every new one, and kept the window repainting at 60 Hz while idle.
			painter.After(gradientFrameInterval)
		}
	}
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

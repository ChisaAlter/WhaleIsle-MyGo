package chrome

import (
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"whaleisle/internal/desktop/wallpapers"
)

var backgroundState struct {
	prefs     wallpapers.Preferences
	active    bool
	wallpaper string
	// colors is the prepared wallpaper's color map the surfaces sample; box is
	// where the wallpaper was drawn and dark the scheme that chose its dim
	// mask. version changes whenever a sampled color could go stale.
	colors  *wallpaperColors
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
	if !backgroundState.active || path == "" {
		// No wallpaper: nothing of one stays referenced.
		backgroundState.colors, backgroundState.hasBox = nil, false
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
			// The decode runs on a media worker; its completion repaints.
			handle := wallpaperRequest(path, p, w, h)
			pic, ok := handle.Picture()
			if !ok || pic.Bitmap == nil {
				return
			}
			s := &backgroundState
			if colors, _ := pic.Extra.(*wallpaperColors); colors != s.colors {
				s.colors = colors
				s.version++
			}
			if bw, bh := pic.Bitmap.Size(); strings.HasPrefix(wallpaperKind(p), "wallpaper/pixelate") {
				// Pixelated at screen pixels already: draw at its own size,
				// centred, so the blocks stay exact.
				painter.Clip(r, 0, func() {
					painter.Image(pic.Bitmap, ui.Rect{X: r.X + (r.W-float32(bw))/2, Y: r.Y + (r.H-float32(bh))/2, W: float32(bw), H: float32(bh)}, ui.FillBox)
				})
			} else {
				painter.Image(pic.Bitmap, r, ui.Cover)
			}
			if radius := wallpaperBlurRadius(p.Blur); radius > 0 {
				wallpaperBlurMaterial(radius).PaintMaterial(painter, r, [4]float32{})
			}
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
		if r.W <= 0 || r.H <= 0 {
			return
		}
		// The ambient gradient is baked once per configuration (size, colors,
		// scheme) into a static texture: no phase ticker, no repaint loop.
		config := gradientConfiguration(p, float64(r.W), float64(r.H), painter.Theme().Dark, reduced)
		if frame := ambientGradientFrames.frozen(config, painter.Now()); frame != nil {
			painter.Image(frame.bitmap, r, ui.FillBox)
			publishGradientBackdrop(frame, r)
		}
	}
}

package chrome

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/widgets"
)

// The contrast floors glass keeps for the text that sits on it, as the web's
// contrast rules ask: body text 4.5:1, and secondary text (labels, hints,
// timestamps) 3:1.
const (
	// TextContrast is the least contrast body text must have on its surface.
	TextContrast = widgets.TextContrast
	// SecondaryContrast is the least contrast secondary text must have.
	SecondaryContrast = widgets.ControlContrast
)

// RelativeLuminance returns how bright a color is, from 0 to 1, as the web's
// contrast rules measure it.
func RelativeLuminance(c ui.Color) float64 { return widgets.RelativeLuminance(c) }

// ContrastRatio returns how far apart two colors are, from 1 to 21, as the
// web's contrast rules measure them.
func ContrastRatio(a, b ui.Color) float64 { return widgets.ContrastRatio(a, b) }

// EnsureContrast returns ink adjusted toward black or white until it reads on
// bg at the body-text floor. Ink that already reads comes back unchanged, and
// so does ink whose surface cannot reach the floor at all.
func EnsureContrast(ink, bg ui.Color) ui.Color {
	return EnsureContrastAt(ink, bg, TextContrast)
}

// EnsureContrastAt is EnsureContrast at an explicit floor. It only ever moves
// ink away from bg: a material that paints its own backdrop uses it to keep the
// theme's text colors readable instead of repainting the palette.
func EnsureContrastAt(ink, bg ui.Color, floor float64) ui.Color {
	if ink.A == 0 || floor <= 1 {
		return ink
	}
	if ContrastRatio(ink, bg) >= floor {
		return ink
	}
	target := ui.RGB(255, 255, 255)
	if ContrastRatio(ui.RGB(0, 0, 0), bg) > ContrastRatio(target, bg) {
		target = ui.RGB(0, 0, 0)
	}
	if ContrastRatio(target, bg) < floor {
		// Neither end can reach the floor on this surface.
		return ink
	}
	lo, hi := float32(0), float32(1)
	best := target
	for range 12 {
		mid := (lo + hi) / 2
		candidate := ink.Mix(target, mid)
		if ContrastRatio(candidate, bg) >= floor {
			best, hi = candidate, mid
		} else {
			lo = mid
		}
	}
	return best
}

// MaterialFill returns the color a surface paints: base's color at the
// material's share of the backdrop. A base that stores an alpha, such as
// --dsw-specific-input-major at 80%, keeps its color while the material decides
// that share, which is how the glass slider drives it in the sheet; raising the
// share towards 1 makes the base opaque, which is how a material becomes solid
// where the backdrop would otherwise eat the text.
func MaterialFill(base ui.Color, alpha float32) ui.Color {
	return ui.RGBA(base.R, base.G, base.B, max(0, min(alpha, 1)))
}

// MaterialSurface returns what material fill f leaves over backdrop.
func MaterialSurface(backdrop, f ui.Color) ui.Color { return f.Over(backdrop) }

// MaterialAlphaFor returns the share of the backdrop material m hides over
// backdrop: MaterialAlpha, raised until ink keeps SecondaryContrast on the
// surface it produces. A material whose token value would leave the theme's
// secondary text below 3:1 frosts more instead, so the surface stays the one
// the theme's colors were designed for; drawing it more solid is the fix rather
// than recoloring text, since the token values match the reference sheets.
func MaterialAlphaFor(backdrop, base ui.Color, m Material, p wallpapers.Preferences, dark bool, ink ui.Color) float32 {
	token := MaterialAlpha(m, p, dark)
	if token <= 0 || ink.A == 0 {
		return token
	}
	if ContrastRatio(ink, MaterialSurface(backdrop, MaterialFill(base, token))) >= SecondaryContrast {
		return token
	}
	lo, hi, best := token, float32(1), float32(1)
	for range 12 {
		mid := (lo + hi) / 2
		if ContrastRatio(ink, MaterialSurface(backdrop, MaterialFill(base, mid))) >= SecondaryContrast {
			best, hi = mid, mid
		} else {
			lo = mid
		}
	}
	return best
}

package chrome

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
)

// Pointer effect bounds and defaults, mirroring ui-theme/src/cursor-fx.ts:
// 100% is the authored look, the sliders clamp to these ends, and an empty
// palette follows the theme accent.
const (
	// MinCursorSpeed is the slowest pointer animation the settings offer.
	MinCursorSpeed = 20
	// MaxCursorSpeed is the fastest pointer animation the settings offer.
	MaxCursorSpeed = 300
	// MinCursorSize is the smallest pointer effect the settings offer.
	MinCursorSize = 25
	// MaxCursorSize is the largest pointer effect the settings offer.
	MaxCursorSize = 300
	// defaultCursorColor keeps the effect visible when no accent is resolved.
	defaultCursorColor = 0x888888
)

// Cursor effect names the renderer draws.
const (
	// CursorEffectTrail stamps fading cells along the pointer path.
	CursorEffectTrail = "trail"
	// CursorEffectSplash mixes a fluid field where the pointer moves and
	// presses.
	CursorEffectSplash = "splash"
)

// CursorEffectConfig maps the stored pointer preferences onto the renderer's
// decoration config. It clamps speed and size to the settings bounds, fills an
// empty palette with the theme accent, and drops the effect under reduce
// motion, which leaves the metallic button sheen in place (static).
//
// The effect is painted by the renderer, over every layer of the window, so the
// shell only has to install this config; there is no per-frame work for the
// view. It reports whether the pointer effect is on.
func CursorEffectConfig(p wallpapers.Preferences, accent ui.Color, reducedMotion bool) (ui.DecorativeEffects, bool) {
	config := ui.DecorativeEffects{
		PointerEnabled:  p.CursorEnabled && !reducedMotion,
		PointerEffect:   cursorEffectName(p.CursorEffect),
		Speed:           clampCursorRange(p.CursorSpeed, MinCursorSpeed, MaxCursorSpeed),
		Size:            clampCursorRange(p.CursorSize, MinCursorSize, MaxCursorSize),
		MetallicButtons: p.MetallicButtons,
	}
	for _, c := range cursorPalette(p.CursorColors, accent) {
		if config.ColorCount >= len(config.Colors) {
			break
		}
		config.Colors[config.ColorCount] = c
		config.ColorCount++
	}
	return config, config.PointerEnabled
}

// ApplyCursorEffects installs the pointer decoration on c, reading reduce
// motion from the desktop preferences. The view root calls it once per frame;
// the renderer paints the trail or splash above every layer of the window.
func ApplyCursorEffects(c *ui.Context, p wallpapers.Preferences, accent ui.Color) bool {
	config, enabled := CursorEffectConfig(p, accent, c.Preferences().ReduceMotion)
	ui.SetDecorativeEffects(c, config)
	return enabled
}

// cursorEffectName resolves the stored effect id; unknown ids fall back to the
// trail the settings ship with.
func cursorEffectName(id string) string {
	if strings.TrimSpace(strings.ToLower(id)) == CursorEffectSplash {
		return CursorEffectSplash
	}
	return CursorEffectTrail
}

// clampCursorRange bounds a stored percent, defaulting a missing one to 100.
func clampCursorRange(v float64, low, high float64) float32 {
	if v <= 0 {
		v = 100
	}
	return float32(max(low, min(high, v)))
}

// cursorPalette returns the stored #rrggbb slots, or the accent when the user
// kept the default palette.
func cursorPalette(stored [6]string, accent ui.Color) []ui.Color {
	out := make([]ui.Color, 0, len(stored))
	for _, raw := range stored {
		raw = strings.TrimSpace(raw)
		if len(raw) != 7 || !strings.HasPrefix(raw, "#") {
			continue
		}
		if _, err := strconv.ParseUint(raw[1:], 16, 24); err != nil {
			continue
		}
		if c, err := parseHexSafe(raw); err == nil {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		if accent.A == 0 {
			return []ui.Color{ui.RGB((defaultCursorColor>>16)&0xff, (defaultCursorColor>>8)&0xff, defaultCursorColor&0xff)}
		}
		return []ui.Color{accent}
	}
	return out
}

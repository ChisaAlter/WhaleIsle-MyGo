package theme

import "whaleisle/internal/desktop/wallpapers"

// FirstRunAppearance is the appearance a profile with no stored appearance
// settings starts with. It mirrors the harness install defaults
// (ui-theme/src/theme-settings.ts DEFAULT_THEME_SETTINGS): the ambient gradient
// runs behind the chrome, the glass slider sits at 80, the terminal at 75, the
// sidebar mask is hidden, no wallpaper is set, and the pointer effects and
// metallic sheen stay off until the user turns them on.
//
// The gradient is the installed build's own backdrop, sampled from
// evidence/compare/ref-hero-1440.png (see wallpapers.PresetColors("dune")): a
// soft full-bleed wash, green at the lower left, lavender at the floor, bright
// in the middle. It replaced the emerald/violet "aurora" blooms, whose blurred
// shadow casts read as hard-edged white circles.
//
// A profile that has saved appearance settings never reads this: the shell
// applies it only where the native appearance key is absent, so an explicit
// choice (including "no backdrop") survives.
func FirstRunAppearance() wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.Effect = "none"
	p.Preset = "dune"
	p.Variant = "orbs"
	p.Colors = wallpapers.PresetColors("dune")
	p.Speed = 190
	p.Count = 5
	return p
}

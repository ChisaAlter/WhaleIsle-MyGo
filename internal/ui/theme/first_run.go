package theme

import "whaleisle/internal/desktop/wallpapers"

// FirstRunAppearance is the appearance a profile with no stored appearance
// settings starts with. It mirrors the harness install defaults
// (ui-theme/src/theme-settings.ts DEFAULT_THEME_SETTINGS): the ambient aurora
// gradient runs behind the chrome, the glass slider sits at 80, the terminal at
// 75, the sidebar mask is hidden, no wallpaper is set, and the pointer effects
// and metallic sheen stay off until the user turns them on.
//
// A profile that has saved appearance settings never reads this: the shell
// applies it only where the native appearance key is absent, so an explicit
// choice (including "no backdrop") survives.
func FirstRunAppearance() wallpapers.Preferences {
	p := wallpapers.Defaults()
	p.Effect = "gradient"
	p.Preset = "aurora"
	p.Variant = "orbs"
	p.Colors = wallpapers.PresetColors("aurora")
	p.Speed = 190
	p.Count = 5
	return p
}

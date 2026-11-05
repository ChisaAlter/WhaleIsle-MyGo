package wallpapers

import "strings"

type Preferences struct {
	SkinWallpaperEnabled bool      `json:"skinWallpaperEnabled"`
	CursorEnabled        bool      `json:"cursorEffectEnabled"`
	CursorEffect         string    `json:"cursorEffect"`
	CursorPreset         string    `json:"cursorEffectPreset"`
	CursorColors         [6]string `json:"cursorEffectColors"`
	CursorSpeed          float64   `json:"cursorEffectSpeed"`
	CursorSize           float64   `json:"cursorEffectSize"`
	MetallicButtons      bool      `json:"metallicPaintEnabled"`
	Sources              []Source  `json:"wallpaperSources"`
	Favorites            []Item    `json:"wallpaperFavorites"`
	Original             string    `json:"wallpaperOriginal"`
	Crop                 Crop      `json:"wallpaperCrop"`
	Blur                 float64   `json:"wallpaperBlur"`
	Pixelate             float64   `json:"wallpaperPixelate"`
	Glass                float64   `json:"glassOpacity"`
	Terminal             float64   `json:"terminalOpacity"`
	Transparent          bool      `json:"transparentTheme"`
	HideSidebarMask      bool      `json:"sidebarMaskHidden"`
	Preset               string    `json:"backgroundEffectPreset"`
	Effect               string    `json:"backgroundEffect"`
	// EffectExplicit records that the user/Host chose backgroundEffect, so an
	// untouched profile stays flat instead of inheriting a seeded gradient.
	EffectExplicit bool      `json:"-"`
	Variant        string    `json:"backgroundEffectVariant"`
	Speed          float64   `json:"backgroundEffectSpeed"`
	Count          float64   `json:"backgroundEffectCount"`
	Colors         [7]string `json:"backgroundEffectColors"`
}

func Defaults() Preferences {
	return Preferences{SkinWallpaperEnabled: true, CursorEffect: "trail", CursorPreset: "default", CursorSpeed: 100, CursorSize: 100, Sources: Sources(), Crop: DefaultCrop(), Glass: 80, Terminal: 75, HideSidebarMask: true, Preset: "default", Effect: "none", Variant: "orbs", Speed: 190, Count: 5, Colors: [7]string{}}
}

// NormalizeEffect resolves the backdrop effect for a profile: without a
// wallpaper image only an explicit choice can paint the ambient gradient; an
// unset or empty effect falls back to the flat theme base.
func NormalizeEffect(path string, p Preferences) Preferences {
	p.Effect = strings.TrimSpace(p.Effect)
	if strings.TrimSpace(path) != "" {
		return p
	}
	if !p.EffectExplicit || p.Effect == "" {
		p.Effect = "none"
	}
	return p
}

func (p Preferences) Clone() Preferences {
	p.Sources = append([]Source(nil), p.Sources...)
	p.Favorites = append([]Item(nil), p.Favorites...)
	return p
}
func PresetColors(id string) [7]string {
	switch id {
	case "dune":
		// The installed build's default backdrop, sampled from
		// evidence/compare/ref-hero-1440.png: a grey-blue wash at the top
		// (#D1D6DC/#CED5DD), green toward the lower left (#B7D4C8, #B2D0CB), a
		// lavender floor (#C6C4DB), a grey-blue right flank (#D0D4DD) and the
		// bright centre (#F4F4F8). Slot 0 and 1 are the wash's top and bottom,
		// 2..6 the blooms, in the order the painter places them.
		return [7]string{"#D1D6DC", "#C6C4DB", "#F4F4F8", "#B7D4C8", "#B2D0CB", "#D0D4DD", "#CED5DD"}
	case "aurora":
		return [7]string{"", "", "#34d399", "#22d3ee", "#a78bfa", "#4ade80", "#38bdf8"}
	case "sunset":
		return [7]string{"", "", "#fb923c", "#fb7185", "#f472b6", "#fbbf24", "#f87171"}
	case "ocean":
		return [7]string{"", "", "#38bdf8", "#2dd4bf", "#60a5fa", "#818cf8", "#22d3ee"}
	case "sakura":
		return [7]string{"", "", "#f9a8d4", "#f472b6", "#c4b5fd", "#fda4af", "#fbcfe8"}
	}
	return [7]string{}
}

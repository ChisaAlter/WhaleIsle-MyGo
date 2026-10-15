package wallpapers

type Preferences struct {
	Sources         []Source  `json:"wallpaperSources"`
	Favorites       []Item    `json:"wallpaperFavorites"`
	Original        string    `json:"wallpaperOriginal"`
	Crop            Crop      `json:"wallpaperCrop"`
	Blur            float64   `json:"wallpaperBlur"`
	Pixelate        float64   `json:"wallpaperPixelate"`
	Glass           float64   `json:"glassOpacity"`
	Terminal        float64   `json:"terminalOpacity"`
	Transparent     bool      `json:"transparentTheme"`
	HideSidebarMask bool      `json:"sidebarMaskHidden"`
	Preset          string    `json:"backgroundEffectPreset"`
	Effect          string    `json:"backgroundEffect"`
	Variant         string    `json:"backgroundEffectVariant"`
	Speed           float64   `json:"backgroundEffectSpeed"`
	Count           float64   `json:"backgroundEffectCount"`
	Colors          [7]string `json:"backgroundEffectColors"`
}

func Defaults() Preferences {
	return Preferences{Sources: Sources(), Crop: DefaultCrop(), Glass: 80, Terminal: 75, HideSidebarMask: true, Preset: "default", Effect: "none", Variant: "orbs", Speed: 190, Count: 5, Colors: [7]string{}}
}

func (p Preferences) Clone() Preferences {
	p.Sources = append([]Source(nil), p.Sources...)
	p.Favorites = append([]Item(nil), p.Favorites...)
	return p
}
func PresetColors(id string) [7]string {
	switch id {
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

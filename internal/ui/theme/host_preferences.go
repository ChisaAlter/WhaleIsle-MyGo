package theme

import (
	"encoding/json"

	"whaleisle/internal/desktop/wallpapers"
)

// HostSettings is the actual ui-theme namespace, distinct from desktop-only
// launcher/window startup config. Missing fields use the original schema defaults.
type HostSettings struct {
	Preference              string              `json:"preference"`
	FontSize                float32             `json:"fontSize"`
	TerminalFontSize        float32             `json:"terminalFontSize"`
	ActiveLightThemeID      string              `json:"activeLightThemeId"`
	ActiveDarkThemeID       string              `json:"activeDarkThemeId"`
	CustomThemes            []Family            `json:"customThemes"`
	GlassOpacity            float64             `json:"glassOpacity"`
	TerminalOpacity         float64             `json:"terminalOpacity"`
	TransparentTheme        bool                `json:"transparentTheme"`
	SidebarMaskHidden       bool                `json:"sidebarMaskHidden"`
	WallpaperImage          string              `json:"wallpaperImage"`
	WallpaperBlur           float64             `json:"wallpaperBlur"`
	WallpaperPixelate       float64             `json:"wallpaperPixelate"`
	WallpaperSources        []wallpapers.Source `json:"wallpaperSources"`
	WallpaperFavorites      []wallpapers.Item   `json:"wallpaperFavorites"`
	BackgroundEffect        string              `json:"backgroundEffect"`
	BackgroundEffectColors  [7]string           `json:"backgroundEffectColors"`
	BackgroundEffectSpeed   float64             `json:"backgroundEffectSpeed"`
	BackgroundEffectCount   float64             `json:"backgroundEffectCount"`
	BackgroundEffectPreset  string              `json:"backgroundEffectPreset"`
	BackgroundEffectVariant string              `json:"backgroundEffectVariant"`
	CursorEffectEnabled     bool                `json:"cursorEffectEnabled"`
	CursorEffect            string              `json:"cursorEffect"`
	CursorEffectColors      [6]string           `json:"cursorEffectColors"`
	CursorEffectSpeed       float64             `json:"cursorEffectSpeed"`
	CursorEffectSize        float64             `json:"cursorEffectSize"`
	CursorEffectPreset      string              `json:"cursorEffectPreset"`
	MetallicPaintEnabled    bool                `json:"metallicPaintEnabled"`
	FontFamilySans          string              `json:"fontFamilySans"`
	FontFamilyCode          string              `json:"fontFamilyCode"`
	FontSizeInterface       float32             `json:"fontSizeInterface"`
	FontSizeCode            float32             `json:"fontSizeCode"`
	FontFamilyComposer      string              `json:"fontFamilyComposer"`
	FontFamilyTerminal      string              `json:"fontFamilyTerminal"`
}

func ReadHostSettings(raw json.RawMessage) (HostSettings, error) {
	h := HostSettings{Preference: "system", FontSize: 14, TerminalFontSize: 13, ActiveLightThemeID: "deepseek", ActiveDarkThemeID: "deepseek", GlassOpacity: 80, TerminalOpacity: 75, SidebarMaskHidden: true, WallpaperSources: wallpapers.Sources(), BackgroundEffect: "gradient", BackgroundEffectColors: wallpapers.PresetColors("aurora"), BackgroundEffectSpeed: 190, BackgroundEffectCount: 5, BackgroundEffectPreset: "aurora", BackgroundEffectVariant: "orbs", CursorEffect: "trail", CursorEffectSpeed: 100, CursorEffectSize: 100, CursorEffectPreset: "default", FontSizeInterface: 16, FontSizeCode: 13}
	err := json.Unmarshal(raw, &h)
	if err == nil {
		h.WallpaperSources = hostWallpaperSources(raw)
		h.WallpaperFavorites = sanitizeHostFavorites(h.WallpaperFavorites)
	}
	return h, err
}

func (h HostSettings) Preferences() Preferences {
	sans, code, terminal := h.FontFamilySans, h.FontFamilyCode, h.FontFamilyTerminal
	return Preferences{Light: h.ActiveLightThemeID, Dark: h.ActiveDarkThemeID, Custom: h.CustomThemes, Interface: FontPreference{Family: sans, Size: h.FontSizeInterface}, Chat: FontPreference{Family: sans, Size: h.FontSize}, Code: FontPreference{Family: code, Size: h.FontSizeCode}, Composer: FontPreference{Family: h.FontFamilyComposer, Size: h.FontSize}, Terminal: FontPreference{Family: terminal, Size: h.TerminalFontSize}}.Clone()
}

func (h HostSettings) Background() wallpapers.Preferences {
	return wallpapers.Preferences{CursorEnabled: h.CursorEffectEnabled, CursorEffect: h.CursorEffect, CursorPreset: h.CursorEffectPreset, CursorColors: h.CursorEffectColors, CursorSpeed: h.CursorEffectSpeed, CursorSize: h.CursorEffectSize, MetallicButtons: h.MetallicPaintEnabled, Sources: append([]wallpapers.Source(nil), h.WallpaperSources...), Favorites: append([]wallpapers.Item(nil), h.WallpaperFavorites...), Crop: wallpapers.DefaultCrop(), Blur: h.WallpaperBlur, Pixelate: h.WallpaperPixelate, Glass: h.GlassOpacity, Terminal: h.TerminalOpacity, Transparent: h.TransparentTheme, HideSidebarMask: h.SidebarMaskHidden, Preset: h.BackgroundEffectPreset, Effect: h.BackgroundEffect, Variant: h.BackgroundEffectVariant, Speed: h.BackgroundEffectSpeed, Count: h.BackgroundEffectCount, Colors: h.BackgroundEffectColors}
}

// TypographyFields contains only source-defined Host fields. Composer has no
// independent size setting; it follows the content size rather than inventing one.
func TypographyFields(p Preferences) map[string]any {
	if p.Custom == nil {
		p.Custom = []Family{}
	}
	return map[string]any{"fontFamilySans": p.Interface.Family, "fontFamilyCode": p.Code.Family, "fontSizeInterface": p.Interface.Size, "fontSizeCode": p.Code.Size, "fontFamilyComposer": p.Composer.Family, "fontFamilyTerminal": p.Terminal.Family, "fontSize": p.Chat.Size, "terminalFontSize": p.Terminal.Size, "activeLightThemeId": p.Light, "activeDarkThemeId": p.Dark, "customThemes": p.Custom}
}

func BackgroundFields(p wallpapers.Preferences) map[string]any {
	p.Sources = sanitizeHostSources(p.Sources)
	p.Favorites = sanitizeHostFavorites(p.Favorites)
	if p.Sources == nil {
		p.Sources = []wallpapers.Source{}
	}
	if p.Favorites == nil {
		p.Favorites = []wallpapers.Item{}
	}
	return map[string]any{"wallpaperBlur": p.Blur, "wallpaperPixelate": p.Pixelate, "glassOpacity": p.Glass, "terminalOpacity": p.Terminal, "transparentTheme": p.Transparent, "sidebarMaskHidden": p.HideSidebarMask, "wallpaperSources": p.Sources, "wallpaperFavorites": p.Favorites, "backgroundEffectPreset": p.Preset, "backgroundEffect": p.Effect, "backgroundEffectVariant": p.Variant, "backgroundEffectSpeed": p.Speed, "backgroundEffectCount": p.Count, "backgroundEffectColors": p.Colors, "cursorEffectEnabled": p.CursorEnabled, "cursorEffect": p.CursorEffect, "cursorEffectPreset": p.CursorPreset, "cursorEffectColors": p.CursorColors, "cursorEffectSpeed": p.CursorSpeed, "cursorEffectSize": p.CursorSize, "metallicPaintEnabled": p.MetallicButtons}
}

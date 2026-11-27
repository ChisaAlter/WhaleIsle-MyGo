package main

import (
	"errors"
	"math"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

type designGlassSignature struct {
	enabled         bool
	backdropActive  bool
	hasWallpaper    bool
	dark            bool
	glass           float64
	transparent     bool
	hideSidebarMask bool
	acrylic         bool
}

var designGlassState struct {
	mu    sync.Mutex
	valid bool
	sig   designGlassSignature
}

func clampGlassPercent(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(0, math.Min(100, v))
}

func canvasUncappedPercent(g float64) float64 {
	k := clampGlassPercent(g)
	switch {
	case k <= 40:
		return math.Round(k * 0.375)
	case k <= 80:
		return math.Round(15 + (k-40)*0.75)
	default:
		return math.Round(45 + (k-80)*2.75)
	}
}

// acrylicCanvasAlpha implements round(100 * (canvasUncapped(g)/100)^1.5) / 100
// so Windows Acrylic at g=80% yields a 0.30 canvas fill.
func acrylicCanvasAlpha(prefs wallpapers.Preferences) float32 {
	if prefs.Transparent {
		return 0
	}
	u := canvasUncappedPercent(prefs.Glass) / 100.0
	return float32(math.Round(100.0*math.Pow(u, 1.5)) / 100.0)
}

// darkGlassMenuAlpha implements max(g*0.60, g*2.60-160)/100
// (g=0% -> 0.00, g=80% -> 0.48, g=100% -> 1.00).
func darkGlassMenuAlpha(prefs wallpapers.Preferences) float32 {
	g := clampGlassPercent(prefs.Glass)
	pct := math.Max(g*0.60, g*2.60-160)
	return float32(math.Max(0, math.Min(100, pct)) / 100.0)
}

// lightGlassMenuAlpha implements max(g*0.55, g*2.80-180)/100
// (g=0% -> 0.00, g=80% -> 0.44, g=100% -> 1.00).
func lightGlassMenuAlpha(prefs wallpapers.Preferences) float32 {
	g := clampGlassPercent(prefs.Glass)
	pct := math.Max(g*0.55, g*2.80-180)
	return float32(math.Max(0, math.Min(100, pct)) / 100.0)
}

func applyDarkGlassPalette(p *design.Palette, canvasA, sidebarA, panelA, menuA float32) {
	p.Background = chrome.MaterialFill(ui.RGB(21, 21, 23), canvasA)
	p.Sidebar = chrome.MaterialFill(ui.RGB(27, 27, 28), sidebarA)
	p.Card = chrome.MaterialFill(ui.RGB(32, 32, 35), panelA)
	p.Popover = chrome.MaterialFill(ui.RGB(38, 39, 43), menuA)
	p.Code = ui.RGB(27, 27, 28)
	p.Border = ui.RGBA(255, 255, 255, 0.07)
	p.SidebarBorder = ui.RGBA(255, 255, 255, 0.055)
	p.Input = ui.RGBA(255, 255, 255, 0.095)
	p.Secondary = ui.RGBA(255, 255, 255, 0.06)
	p.Muted = ui.RGBA(255, 255, 255, 0.05)
	p.Accent = ui.RGBA(255, 255, 255, 0.075)
	p.SidebarAccent = ui.RGBA(255, 255, 255, 0.08)
	p.CodeHighlight = ui.RGBA(255, 255, 255, 0.06)
	p.Foreground = ui.RGB(245, 246, 248)
	p.PopoverForeground = ui.RGB(245, 246, 248)
	p.MutedForeground = ui.RGB(173, 178, 184)
	p.SidebarForeground = ui.RGB(212, 216, 220)
}

func applyLightGlassPalette(p *design.Palette, prefs wallpapers.Preferences) {
	panelAlpha := chrome.MaterialAlpha(chrome.MaterialPanel, prefs, false)
	p.Background = chrome.MaterialFill(p.Background, chrome.MaterialAlpha(chrome.MaterialCanvas, prefs, false))
	p.Sidebar = chrome.MaterialFill(ui.RGB(249, 250, 251), chrome.MaterialAlpha(chrome.MaterialSidebar, prefs, false))
	p.Card = chrome.MaterialFill(ui.RGB(252, 253, 255), panelAlpha)
	p.Popover = chrome.MaterialFill(ui.RGB(252, 253, 254), lightGlassMenuAlpha(prefs))
	p.Code = ui.RGB(249, 250, 251)
	p.Border = ui.RGBA(0, 0, 0, 0.075)
	p.SidebarBorder = ui.RGBA(0, 0, 0, 0.055)
	p.Input = ui.RGBA(0, 0, 0, 0.095)
	p.Secondary = ui.RGBA(0, 0, 0, 0.055)
	p.Muted = ui.RGBA(0, 0, 0, 0.055)
	p.Accent = ui.RGBA(0, 0, 0, 0.055)
	p.SidebarAccent = ui.RGBA(0, 0, 0, 0.055)
	p.CodeHighlight = ui.RGBA(0, 0, 0, 0.055)
	p.Foreground = ui.RGB(17, 19, 24)
	p.PopoverForeground = ui.RGB(17, 19, 24)
	p.MutedForeground = ui.RGB(52, 57, 66)
	p.SidebarForeground = ui.RGB(36, 41, 48)
}

func glassSurfaceFilter(sig designGlassSignature) design.SurfaceFilter {
	prefs := wallpapers.Defaults()
	prefs.Glass = sig.glass
	prefs.Transparent = sig.transparent
	prefs.HideSidebarMask = sig.hideSidebarMask
	return func(s design.Scheme, p *design.Palette) {
		dark := s == design.Dark
		if sig.backdropActive {
			if dark {
				applyDarkGlassPalette(
					p,
					chrome.MaterialAlpha(chrome.MaterialCanvas, prefs, true),
					chrome.MaterialAlpha(chrome.MaterialSidebar, prefs, true),
					chrome.MaterialAlpha(chrome.MaterialPanel, prefs, true),
					darkGlassMenuAlpha(prefs),
				)
			} else {
				applyLightGlassPalette(p, prefs)
			}
			return
		}
		if dark && sig.acrylic {
			sidebarA := chrome.ShowThroughAlpha(prefs, true)
			if sig.hideSidebarMask {
				sidebarA = 0
			}
			applyDarkGlassPalette(
				p,
				acrylicCanvasAlpha(prefs),
				sidebarA,
				0.80,
				darkGlassMenuAlpha(prefs),
			)
		}
	}
}

// syncDesignGlass updates design.SetGlass only when the effective backdrop
// signature changes, avoiding per-frame token cache resets and revision bumps.
func syncDesignGlass(backdropActive, hasWallpaper, dark, acrylic, skinFlat bool, prefs wallpapers.Preferences) {
	enabled := !renderpolicy.NoBackdrop() && !skinFlat && (backdropActive || acrylic)
	var sig designGlassSignature
	if enabled {
		eff := prefs
		if eff.Transparent && !hasWallpaper {
			eff.Transparent = false
		}
		sig = designGlassSignature{
			enabled:         true,
			backdropActive:  backdropActive,
			hasWallpaper:    hasWallpaper,
			dark:            dark,
			glass:           eff.Glass,
			transparent:     eff.Transparent,
			hideSidebarMask: eff.HideSidebarMask,
			acrylic:         acrylic,
		}
	}
	designGlassState.mu.Lock()
	changed := !designGlassState.valid || designGlassState.sig != sig || design.GlassEnabled() != sig.enabled
	if changed {
		designGlassState.valid = true
		designGlassState.sig = sig
	}
	designGlassState.mu.Unlock()
	if !changed {
		return
	}
	if !sig.enabled {
		design.SetGlass(false, nil)
		return
	}
	design.SetGlass(true, glassSurfaceFilter(sig))
}

func syncDesignGlassFromPolicy() {
	if renderpolicy.NoBackdrop() {
		syncDesignGlass(false, false, false, false, false, wallpapers.Defaults())
		return
	}
	p := theme.CurrentPreferences()
	systemDark := false
	if mygo.App.IsReady() {
		systemDark = mygo.Theme.IsDark()
	}
	skinFlat := theme.SkinForMode(p, systemDark) == theme.SkinCodex
	prefs := chrome.BackdropPreferences()
	path := ""
	if chrome.BackdropHasWallpaper() {
		path = "wallpaper"
	}
	dark := chrome.BackdropDark()
	acrylic := !skinFlat && desktopGlassMaterial(path, prefs, dark, false) != mygo.VibrancyNone
	syncDesignGlass(chrome.BackdropActive(), chrome.BackdropHasWallpaper(), dark, acrylic, skinFlat, prefs)
}

func (s *mainState) syncAppearanceBackdrop(id string, systemDark, dark bool) {
	skinFlat := theme.ResolvedSkin(id, systemDark) == theme.SkinCodex
	if renderpolicy.NoBackdrop() {
		prefs := wallpapers.Defaults()
		if g := s.wallpaperEditor(); g != nil {
			prefs = g.EffectivePreferences()
		}
		prefs = renderpolicy.Background(prefs)
		chrome.ConfigureBackdrop("", prefs)
		chrome.SetBackdropDark(dark)
		syncDesignGlass(false, false, dark, false, skinFlat, prefs)
		return
	}
	if skinFlat {
		prefs := wallpapers.Defaults()
		if g := s.wallpaperEditor(); g != nil {
			prefs = g.EffectivePreferences()
		}
		path, prefs := skinBackdrop(theme.SkinCodex, "", prefs)
		chrome.ConfigureBackdrop(path, prefs)
		chrome.SetBackdropDark(dark)
		syncDesignGlass(false, false, dark, false, true, prefs)
		return
	}
	path := s.resolvedWallpaperPath(s.currentWallpaperConfigPath())
	prefs := wallpapers.Defaults()
	if g := s.wallpaperEditor(); g != nil {
		prefs = g.EffectivePreferences()
	}
	prefs = wallpaperBackdropPrefs(path, prefs)
	chrome.ConfigureBackdrop(path, prefs)
	chrome.SetBackdropDark(dark)
	acrylic := desktopGlassMaterial(path, prefs, dark, false) != mygo.VibrancyNone
	syncDesignGlass(chrome.BackdropActive(), strings.TrimSpace(path) != "", dark, acrylic, false, prefs)
}

// applyGlassTheme puts the optional wallpaper-and-glass theme in force for
// every window: with it off, the backdrop painter configures no wallpaper,
// gradient or glass sampling at all.
func applyGlassTheme(on bool) {
	renderpolicy.SetGlass(on)
	syncDesignGlassFromPolicy()
}

var errGlassThemeUnavailable = errors.New("glass theme: the settings store is not open yet")

// migrateGlassTheme stores the effective switch once, so a profile from
// before the switch keeps the decision its own wallpaper choice implied.
func migrateGlassTheme(store *config.Store) bool {
	cfg := store.Get()
	on := cfg.GlassThemeOn()
	if cfg.GlassTheme == nil {
		_ = store.Update(func(c *config.Config) { v := on; c.GlassTheme = &v })
	}
	return on
}

// setGlassTheme applies the switch optimistically in memory and persists it
// asynchronously on the single-writer debounced config queue.
func (s *mainState) setGlassTheme(on bool) error {
	if s.cfgStore == nil {
		return errGlassThemeUnavailable
	}
	applyGlassTheme(on)
	s.enqueueConfigUpdate(s.cfgStore, func(c *config.Config) { v := on; c.GlassTheme = &v })
	s.invalidateAppearance()
	return nil
}

// glassThemeAccess is the appearance page's glass-theme switch.
func (s *mainState) glassThemeAccess() *settings.GlassThemeAccess {
	if s.cfgStore == nil {
		return nil
	}
	enabled := s.effectiveConfigFor(s.cfgStore).GlassThemeOn()
	return &settings.GlassThemeAccess{Enabled: &enabled, OnToggle: s.setGlassTheme}
}

package main

import (
	"errors"
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

func applyDarkGlassPalette(p *design.Palette, canvasA, sidebarA, panelA, menuA float32) {
	p.Background = chrome.MaterialFill(ui.RGB(18, 19, 23), canvasA)
	p.Sidebar = chrome.MaterialFill(ui.RGB(22, 24, 30), sidebarA)
	p.Card = chrome.MaterialFill(ui.RGB(30, 32, 38), panelA)
	p.Popover = chrome.MaterialFill(ui.RGB(34, 36, 44), menuA)
	p.Code = chrome.MaterialFill(ui.RGB(30, 32, 38), panelA)
	p.Border = ui.RGBA(255, 255, 255, 0.14)
	p.SidebarBorder = ui.RGBA(255, 255, 255, 0.10)
	p.Input = ui.RGBA(255, 255, 255, 0.16)
	p.Secondary = ui.RGBA(255, 255, 255, 0.10)
	p.Muted = ui.RGBA(255, 255, 255, 0.09)
	p.Accent = ui.RGBA(255, 255, 255, 0.11)
	p.SidebarAccent = ui.RGBA(255, 255, 255, 0.10)
	p.CodeHighlight = ui.RGBA(255, 255, 255, 0.09)
}

func applyLightGlassPalette(p *design.Palette, prefs wallpapers.Preferences) {
	panelAlpha := chrome.MaterialAlpha(chrome.MaterialPanel, prefs, false)
	p.Background = chrome.MaterialFill(p.Background, chrome.MaterialAlpha(chrome.MaterialCanvas, prefs, false))
	p.Sidebar = chrome.MaterialFill(ui.RGB(246, 248, 252), chrome.MaterialAlpha(chrome.MaterialSidebar, prefs, false))
	p.Card = chrome.MaterialFill(ui.RGB(252, 253, 255), panelAlpha)
	p.Popover = chrome.MaterialFill(ui.RGB(252, 253, 255), chrome.MaterialAlpha(chrome.MaterialMenu, prefs, false))
	p.Code = chrome.MaterialFill(ui.RGB(252, 253, 255), panelAlpha)
	p.Border = ui.RGBA(0, 0, 0, 0.09)
	p.SidebarBorder = ui.RGBA(0, 0, 0, 0.07)
	p.Input = ui.RGBA(0, 0, 0, 0.12)
	p.Secondary = ui.RGBA(0, 0, 0, 0.055)
	p.Muted = ui.RGBA(0, 0, 0, 0.055)
	p.Accent = ui.RGBA(0, 0, 0, 0.055)
	p.SidebarAccent = ui.RGBA(0, 0, 0, 0.055)
	p.CodeHighlight = ui.RGBA(0, 0, 0, 0.055)
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
					chrome.MaterialAlpha(chrome.MaterialMenu, prefs, true),
				)
			} else {
				applyLightGlassPalette(p, prefs)
			}
			return
		}
		if dark && sig.acrylic {
			alpha := chrome.ShowThroughAlpha(prefs, true)
			applyDarkGlassPalette(p, alpha, alpha, alpha, alpha)
		}
	}
}

// syncDesignGlass updates design.SetGlass only when the effective backdrop
// signature changes, avoiding per-frame token cache resets and revision bumps.
func syncDesignGlass(backdropActive, hasWallpaper, dark, acrylic, skinFlat bool, prefs wallpapers.Preferences) {
	enabled := !renderpolicy.NoBackdrop() && !skinFlat
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
	syncDesignGlass(chrome.BackdropActive(), chrome.BackdropHasWallpaper(), chrome.BackdropDark(), false, skinFlat, chrome.BackdropPreferences())
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

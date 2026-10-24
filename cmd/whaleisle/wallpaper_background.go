package main

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"log"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) wallpaperEditor() *settings.WallpaperEditor {
	if s.wallpaperControls != nil {
		return s.wallpaperControls
	}
	if s.cfgStore == nil {
		return nil
	}
	p, err := s.cfgStore.WallpaperBackground()
	// A fresh profile starts on the product's dark appearance with the default
	// aurora background, matching the shipped Electron first-run look. Saved
	// profiles keep whatever they already stored.
	if !s.cfgStore.HasWallpaperBackground() {
		p = theme.FirstRunAppearance()
	}
	g := settings.NewWallpaperEditor(p, filepath.Join(paths.UserDataDir(), "wallpapers"), func(p wallpapers.Preferences) error {
		if s.hostAppearanceAccepted {
			if s.hostAppearance == nil {
				return fmt.Errorf("Host appearance is unavailable")
			}
			return s.hostAppearance.SaveBackground(p)
		}
		return s.cfgStore.SaveWallpaperBackground(p)
	}, func(path string) error {
		if s.hostAppearanceAccepted {
			if s.hostAppearance == nil {
				return fmt.Errorf("Host appearance is unavailable")
			}
			return s.hostAppearance.ApplyWallpaper(path)
		}
		if err := s.cfgStore.Update(func(cfg *config.Config) { cfg.WallpaperPath = path }); err != nil {
			return err
		}
		s.mu.Lock()
		s.desktopDraft.wallpaperPath = path
		s.mu.Unlock()
		return nil
	}, func(task func() func()) {
		go func() {
			done := task()
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			if win != nil {
				win.Update(done)
			}
		}()
	})
	g.Pending = s.hostAppearancePending
	if err != nil {
		g.Status = err.Error()
	}
	s.wallpaperControls = g
	return g
}
func (s *mainState) wallpaperSettings(visible bool) *settings.WallpaperEditor {
	s.wallpaperVisible = visible
	return s.wallpaperEditor()
}
func (s *mainState) beginWallpaperFrame() { s.wallpaperVisible = false }
func (s *mainState) endWallpaperFrame() {
	if !s.wallpaperVisible && s.wallpaperControls != nil {
		s.wallpaperControls.Prefs = s.wallpaperControls.Saved.Clone()
	}
}

// wallpaperSourcePath picks the image the window backdrop paints: an accepted
// Host appearance owns the value only when the Host actually has a wallpaper;
// otherwise MyGo's own goWallpaperPath (the seeded-fixture field) stays in
// force. Blindly preferring the Host value dropped a seeded config wallpaper as
// soon as an empty Host theme arrived.
func wallpaperSourcePath(hostAccepted bool, hostImage, configPath string) string {
	if hostAccepted {
		if image := strings.TrimSpace(hostImage); image != "" {
			return image
		}
	}
	return strings.TrimSpace(configPath)
}

// wallpaperBackdropPrefs keeps an image-less backdrop flat. The native profile
// stores wallpaperImage: "" and the page then shows the theme base colour, so
// the ambient aurora must not paint when no image is set; a seeded or Host
// image keeps the stored effect.
func wallpaperBackdropPrefs(path string, prefs wallpapers.Preferences) wallpapers.Preferences {
	return wallpapers.NormalizeEffect(path, prefs)
}

func (s *mainState) resolvedWallpaperPath(path string) string {
	return wallpaperSourcePath(s.hostAppearanceAccepted, s.hostWallpaperImage, path)
}

// wallpaperCanvasFill is the colour painted under everything else. Dark, no
// wallpaper and no explicit gradient sits under the native bg-mask-1, so the
// canvas reads ~12,12,12 instead of the raw 21,21,23 token; panels keep their
// own tokens because only this canvas is masked.
func wallpaperCanvasFill(base ui.Color, dark bool, path string, prefs wallpapers.Preferences) ui.Color {
	prefs = wallpapers.NormalizeEffect(path, prefs)
	if !dark || strings.TrimSpace(path) != "" || prefs.Effect == "gradient" {
		return base
	}
	return chrome.FlatBackdropFill(base, true)
}

func (s *mainState) wallpaperPainter(path string, visible bool, reducedMotion ...bool) func(*ui.Painter, ui.Rect) {
	path = s.resolvedWallpaperPath(path)
	if os.Getenv("WHALEISLE_BACKDROP_PROBE") != "" {
		if g := s.wallpaperEditor(); g != nil {
			log.Printf("backdrop probe: path=%q effect=%q preset=%q colors0=%q explicit=%v active=%v", path, g.Prefs.Effect, g.Prefs.Preset, g.Prefs.Colors[0], g.Prefs.EffectExplicit, chrome.BackdropActive())
		}
	}
	g := s.wallpaperEditor()
	if g == nil {
		return func(*ui.Painter, ui.Rect) {}
	}
	if !visible {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		return func(*ui.Painter, ui.Rect) {}
	}
	return chrome.WallpaperBackgroundPainter(path, wallpaperBackdropPrefs(path, g.Prefs), reducedMotion...)
}

// Existing local-dropdown selections also become durable app-owned copies on save.
func (s *mainState) materializeWallpaperDraft() error {
	if s.hostAppearanceAccepted {
		if s.hostAppearance == nil {
			return fmt.Errorf("Host appearance is unavailable")
		}
		return s.hostAppearance.ApplyWallpaper(s.desktopDraft.wallpaperPath)
	}
	path := s.desktopDraft.wallpaperPath
	if path == "" {
		return nil
	}
	dir := filepath.Join(paths.UserDataDir(), "wallpapers")
	if filepath.Dir(path) == dir {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	copyPath, err := wallpapers.Import(dir, data)
	if err != nil {
		return err
	}
	s.desktopDraft.wallpaperPath = copyPath
	if g := s.wallpaperEditor(); g != nil {
		g.Prefs.Original = copyPath
		g.Prefs.Crop = wallpapers.DefaultCrop()
		if err = g.Save(g.Prefs); err != nil {
			return err
		}
		g.Saved = g.Prefs.Clone()
	}
	return nil
}

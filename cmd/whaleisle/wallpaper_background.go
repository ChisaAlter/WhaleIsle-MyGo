package main

import (
	"fmt"
	"github.com/egoist/mygo/ui"
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
	if !s.cfgStore.HasWallpaperBackground() || s.cfgStore.IsFirstRun() {
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

func (s *mainState) wallpaperPainter(path string, visible bool, reducedMotion ...bool) func(*ui.Painter, ui.Rect) {
	path = wallpaperSourcePath(s.hostAppearanceAccepted, s.hostWallpaperImage, path)
	g := s.wallpaperEditor()
	if g == nil {
		return func(*ui.Painter, ui.Rect) {}
	}
	if !visible {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		return func(*ui.Painter, ui.Rect) {}
	}
	return chrome.WallpaperBackgroundPainter(path, g.Prefs, reducedMotion...)
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

package main

import (
	"fmt"
	"github.com/egoist/mygo"
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

// flatDarkCanvasPreview reports whether the window itself should be opaque with
// the masked canvas: dark scheme, no wallpaper and no explicit gradient.
func flatDarkCanvasPreview(cfgStore *config.Store, cfg config.Config, th theme.Theme) bool {
	dark := th.Scheme == theme.SchemeDark || (th.Scheme != theme.SchemeLight && mygo.Theme.IsDark())
	if !dark || strings.TrimSpace(cfg.WallpaperPath) != "" {
		return false
	}
	if cfgStore != nil {
		if prefs, err := cfgStore.WallpaperBackground(); err == nil {
			return wallpapers.NormalizeEffect("", prefs).Effect != "gradient"
		}
	}
	return true
}

// wallpaperCanvasFill is the colour painted under everything else. Dark, no
// wallpaper and no explicit gradient sits under the native bg-mask-1, so the
// canvas reads ~12,12,12 instead of the raw 21,21,23 token; panels keep their
// own tokens because only this canvas is masked.
// wallpaperCanvasFlatFor decides the flat masked canvas from the stored config
// only, so the render route does not depend on the settings editor being built.
func (s *mainState) wallpaperCanvasFlatFor(path string) bool {
	if strings.TrimSpace(s.resolvedWallpaperPath(path)) != "" {
		return false
	}
	prefs := wallpapers.Defaults()
	if s.cfgStore != nil {
		if stored, err := s.cfgStore.WallpaperBackground(); err == nil {
			prefs = stored
		}
	}
	return wallpapers.NormalizeEffect("", prefs).Effect != "gradient"
}

func appendBackdropLog(line string) {
	if file := os.Getenv("WHALEISLE_BACKDROP_LOG"); file != "" {
		if f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintln(f, line)
			f.Close()
		}
	}
}

func maskedCanvasHex(hex string) string {
	c := chrome.FlatBackdropFill(chrome.Color(hex), true)
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

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
			norm := wallpapers.NormalizeEffect(path, g.Prefs)
			line := fmt.Sprintf("backdrop probe: path=%q effect=%q normalized=%q preset=%q colors0=%q explicit=%v active=%v canvasFlat=%d scheme=%q", path, g.Prefs.Effect, norm.Effect, g.Prefs.Preset, g.Prefs.Colors[0], g.Prefs.EffectExplicit, chrome.BackdropActive(), wallpaperCanvasFill(chrome.Color("#151517"), true, path, g.Prefs).R, s.cfgStore.Get().Theme)
			log.Print(line)
			if file := os.Getenv("WHALEISLE_BACKDROP_LOG"); file != "" {
				if f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
					fmt.Fprintln(f, line)
					f.Close()
				}
			}
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
	prefs := g.Prefs
	// The Host appearance may carry a gradient seed; only a locally stored
	// backgroundEffect counts as the user choice, so a dark profile without a
	// wallpaper stays flat instead of painting the host wash (round239).
	if path == "" && s.cfgStore != nil {
		if local, err := s.cfgStore.WallpaperBackground(); err == nil && !local.EffectExplicit {
			prefs.Effect = "none"
			prefs.EffectExplicit = false
		}
	}
	prefs = wallpaperBackdropPrefs(path, prefs)
	appendBackdropLog(fmt.Sprintf("painter: path=%q effect=%q localExplicit=%v active=%v", path, prefs.Effect, g.Prefs.EffectExplicit, chrome.BackdropActive()))
	return chrome.WallpaperBackgroundPainter(path, prefs, reducedMotion...)
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

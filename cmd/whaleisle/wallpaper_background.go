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
	"whaleisle/internal/ui/i18n"
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
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostAppearanceUnavailable))
			}
			return s.hostAppearance.SaveBackground(p)
		}
		return s.cfgStore.SaveWallpaperBackground(p)
	}, func(path string) error {
		if s.hostAppearanceAccepted {
			if s.hostAppearance == nil {
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostAppearanceUnavailable))
			}
			// Host owns the image and its optimistic revision. Do not publish
			// a local config mirror before that write is accepted.
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
	g.ImageSource = func() string { return s.resolvedWallpaperPath(s.desktopDraft.wallpaperPath) }
	g.EffectiveImage = func(p wallpapers.Preferences) string {
		return s.resolvedWallpaperPathFor(s.desktopDraft.wallpaperPath, p)
	}
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
		s.wallpaperControls.Hide()
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

// wallpaperBackdropPrefs preserves a resolved Host or saved local effect;
// an unset local effect keeps the flat fallback.
func wallpaperBackdropPrefs(path string, prefs wallpapers.Preferences) wallpapers.Preferences {
	return wallpapers.NormalizeEffect(path, prefs)
}

func (s *mainState) resolvedWallpaperPath(path string) string {
	prefs := wallpapers.Defaults()
	if g := s.wallpaperControls; g != nil {
		prefs = g.EffectivePreferences()
	}
	return s.resolvedWallpaperPathFor(path, prefs)
}

func (s *mainState) resolvedWallpaperPathFor(path string, prefs wallpapers.Preferences) string {
	if s.hostAppearanceAccepted && s.hostAppearance != nil && s.hostAppearance.Snapshot().WallpaperImageExplicit {
		path = s.hostWallpaperImage
	} else {
		path = wallpaperSourcePath(s.hostAppearanceAccepted, s.hostWallpaperImage, path)
	}
	if strings.TrimSpace(path) == "" {
		p := theme.RenderPreferences()
		// Theme.IsDark waits for the native app loop. Render-policy callers can
		// run before that loop exists, just like activeSkin's headless path.
		systemDark := false
		if mygo.App.IsReady() {
			systemDark = mygo.Theme.IsDark()
		}
		dark := theme.ResolveWithSystem(s.themeID, systemDark).Scheme == theme.SchemeDark
		id := p.Light
		if dark {
			id = p.Dark
		}
		if prefs.SkinWallpaperEnabled {
			path = p.Family(id).Wallpaper
		}
	}
	return path
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
	if s.activeSkin() == theme.SkinCodex {
		return false
	}
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

// appendBackdropLog formats only when the probe log is enabled: its callers run
// on every frame and the wallpaper path may be a long data URL.
func appendBackdropLog(format string, args ...any) {
	if file := os.Getenv("WHALEISLE_BACKDROP_LOG"); file != "" {
		if f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintf(f, format+"\n", args...)
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
	if s.activeSkin() == theme.SkinCodex {
		prefs := wallpapers.Defaults()
		if g := s.wallpaperEditor(); g != nil {
			prefs = g.EffectivePreferences()
		}
		path, prefs = skinBackdrop(theme.SkinCodex, "", prefs)
		return s.desktopGlassPainter(path, prefs, chrome.WallpaperBackgroundPainter(path, prefs, reducedMotion...))
	}
	path = s.resolvedWallpaperPath(path)
	if os.Getenv("WHALEISLE_BACKDROP_PROBE") != "" {
		if g := s.wallpaperEditor(); g != nil {
			norm := wallpapers.NormalizeEffect(path, g.EffectivePreferences())
			line := fmt.Sprintf("backdrop probe: path=%q effect=%q normalized=%q preset=%q colors0=%q explicit=%v active=%v canvasFlat=%d scheme=%q", path, g.Saved.Effect, norm.Effect, g.Saved.Preset, g.Saved.Colors[0], g.Saved.EffectExplicit, chrome.BackdropActive(), wallpaperCanvasFill(chrome.Color("#151517"), true, path, g.Saved).R, s.cfgStore.Get().Theme)
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
		return s.desktopGlassPainter("", wallpapers.Preferences{Effect: "gradient", EffectExplicit: true}, func(*ui.Painter, ui.Rect) {})
	}
	prefs := wallpaperBackdropPrefs(path, g.EffectivePreferences())
	appendBackdropLog("painter: path=%q effect=%q localExplicit=%v active=%v", path, prefs.Effect, g.Saved.EffectExplicit, chrome.BackdropActive())
	return s.desktopGlassPainter(path, prefs, chrome.WallpaperBackgroundPainter(path, prefs, reducedMotion...))
}

// Existing local-dropdown selections also become durable app-owned copies on save.
func (s *mainState) materializeWallpaperDraft() error {
	if s.hostAppearanceAccepted {
		if s.hostAppearance == nil {
			return fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostAppearanceUnavailable))
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

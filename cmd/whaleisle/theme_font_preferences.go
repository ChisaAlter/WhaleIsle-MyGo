package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"os"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) themeFontEditor(visible bool) *theme.Editor {
	s.appearanceVisible = visible
	if s.appearanceEditor == nil && s.cfgStore != nil {
		p, err := s.cfgStore.Appearance()
		s.appearanceEditor = theme.NewEditor(p, s.saveAppearancePreferences)
		s.appearanceEditor.Pending = s.hostAppearancePending
		s.appearanceEditor.FlushWrites = func() {
			if s.hostAppearanceAccepted && s.hostAppearance != nil {
				s.hostAppearance.Flush()
			}
		}
		s.appearanceEditor.OnImport = func() (string, error) {
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: "Import theme JSON", Filters: []mygo.FileFilter{{Name: "Theme JSON", Extensions: []string{"json"}}}})
			if err != nil || len(paths) == 0 {
				return "", err
			}
			data, err := os.ReadFile(paths[0])
			return string(data), err
		}
		if err != nil {
			s.appearanceEditor.Status = err.Error()
		}
	}
	return s.appearanceEditor
}

func (s *mainState) hostAppearancePending() bool {
	return s.hostAppearance != nil && s.hostAppearance.Pending()
}

func (s *mainState) saveAppearancePreferences(p theme.Preferences) error {
	if s.hostAppearanceAccepted {
		if s.hostAppearance == nil {
			return fmt.Errorf("Host appearance is unavailable")
		}
		return s.hostAppearance.SaveTypography(p)
	}
	return s.cfgStore.SaveAppearance(p)
}

// syncHostAppearance runs on the UI thread before mainView takes s.mu. Accepted
// Host values are shared by the editor and renderer. A queued gesture keeps its
// immediate preview until settlement; a disconnected Host never falls back to
// writing launcher config after this installation has adopted Host authority.
func (s *mainState) syncHostAppearance() {
	if s.settings != nil {
		s.settings.EnsureLoaded(context.Background())
	}
	b := settings.BindHostTheme(s.settings)
	if b == nil {
		return
	}
	s.hostAppearance = b
	h := b.Snapshot()
	if !h.Ready {
		return
	}
	s.hostAppearanceAccepted = true
	if h.Pending {
		return
	}
	ownerChanged := s.hostAppearanceOwner != h.Owner
	s.hostAppearanceOwner = h.Owner
	e := s.themeFontEditor(s.appearanceVisible)
	if e != nil {
		e.OnSave = s.saveAppearancePreferences
		if !h.Writable {
			e.OnSave = nil
		}
		e.AcceptHost(h.Preferences, ownerChanged, h.Error)
	}
	if e == nil || !e.Preview {
		theme.SetPreferences(h.Preferences)
	}
	s.mu.Lock()
	previousScheme := s.themeID
	s.hostWallpaperImage = h.WallpaperImage
	s.desktopDraft.wallpaperPath = h.WallpaperImage
	switch h.Scheme {
	case "light":
		s.themeID = "deepseek.light"
	case "dark":
		s.themeID = "deepseek.dark"
	default:
		s.themeID = "deepseek"
	}
	s.desktopDraft.theme = s.themeID
	acceptedScheme := s.themeID
	s.mu.Unlock()
	if previousScheme != acceptedScheme {
		applyAppearance(acceptedScheme)
	}
	if g := s.wallpaperEditor(); g != nil {
		if ownerChanged {
			g.RetireOwner()
		}
		g.Inline = true
		g.Saved = h.Background.Clone()
		if !g.Busy && !g.Cropping() {
			g.Prefs = h.Background.Clone()
		}
		if h.Error != "" {
			g.Status = h.Error
		}
	}
}
func (s *mainState) beginAppearanceFrame(c *ui.Context) {
	s.appearanceVisible = false
	s.mu.Lock()
	id := s.themeID
	s.mu.Unlock()
	p := theme.RenderPreferences()
	colors := theme.Resolve(id).Active(mygo.Theme.IsDark())
	applyAppearanceTheme(c, p, colors)
}

func applyAppearanceTheme(c *ui.Context, p theme.Preferences, colors theme.Colors) {
	t := *c.Theme()
	t.Font = theme.TextFamily(p.Interface.Family)
	t.FontSize = p.Interface.Size
	t.Background = chrome.Color(colors.BG)
	t.Text = chrome.Color(colors.FG)
	t.TextMuted = chrome.Color(colors.Muted)
	t.Surface = chrome.Color(colors.Field)
	t.SurfaceHover = chrome.Color(colors.Hover)
	t.SurfacePressed = chrome.Color(colors.Pressed)
	t.Border = chrome.Color(colors.Border)
	t.Accent = chrome.Color(colors.Accent)
	t.AccentHover = chrome.Color(colors.ButtonInfoHover)
	t.AccentPressed = chrome.Color(colors.ButtonInfoHover)
	t.AccentText = chrome.Color(colors.ButtonFG)
	t.Focus = t.Accent
	t.Selection = t.Accent.Alpha(.3)
	// ui-theme/src/styles/scrollbar.css uses a 5 DIP thumb on desktop.
	t.ScrollbarWidth = 5
	c.SetTheme(&t)
}
func (s *mainState) endAppearanceFrame() {
	if !s.appearanceVisible && s.appearanceEditor != nil && s.appearanceEditor.Preview {
		s.appearanceEditor.Cancel()
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}
}

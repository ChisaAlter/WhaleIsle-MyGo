package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"os"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
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
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: s.win, Title: i18n.T(i18n.KeyCmdImportThemeJSON), Filters: []mygo.FileFilter{{Name: i18n.T(i18n.KeyCmdThemeJSONFilter), Extensions: []string{"json"}}}})
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
			return fmt.Errorf("%s", i18n.T(i18n.KeyCmdHostAppearanceUnavailable))
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
	// Only a Host document that carries wallpaperImage can override the local
	// wallpaper; a seeded empty/default value keeps the flat base.
	if h.WallpaperImageExplicit {
		s.hostWallpaperImage = h.WallpaperImage
		if h.WallpaperImage != "" {
			s.desktopDraft.wallpaperPath = h.WallpaperImage
		}
	}
	// The Host always reports a scheme (default "system"), so it cannot tell an
	// explicit choice from its default: the configured theme wins until the user
	// actually changes appearance, then the persisted preference wins.
	acceptedScheme := s.themeID
	if s.cfgStore != nil {
		acceptedScheme = startupTheme(s.cfgStore.Get(), h.Scheme, true)
	}
	s.themeID = acceptedScheme
	s.desktopDraft.theme = s.themeID
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
	systemDark := mygo.Theme.IsDark()
	resolved := theme.ResolveWithSystem(id, systemDark)
	dark := resolved.Scheme == theme.SchemeDark
	p.Interface = theme.FontForSkin(p, theme.SkinForMode(p, dark), theme.SkinInterfaceFont)
	colors := resolved.Active(systemDark)
	applyAppearanceTheme(c, p, colors, dark)
}

func applyAppearanceTheme(c *ui.Context, p theme.Preferences, colors theme.Colors, dark bool) {
	t := *c.Theme()
	t.Dark = dark
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
	// ui-theme/src/styles/scrollbar.css uses a 5 DIP thumb; its colors come from
	// the theme's scroll bar tokens (--dsw-alias-scrollbar-bg-l1/-hover-l1), which
	// theme.ScrollbarColors carries as the resolved neutral scale entries.
	t.ScrollbarWidth = 5
	// The reference's scrollbar.css paints the thumb only while its container is
	// hovered or dragged; an always-on track left a bar down the sidebar and every
	// list, which the installed build does not show.
	t.ScrollbarAlways = false
	rest, hover := theme.ScrollbarColors(dark)
	t.Scrollbar, t.ScrollbarHover = chrome.Color(rest), chrome.Color(hover)
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

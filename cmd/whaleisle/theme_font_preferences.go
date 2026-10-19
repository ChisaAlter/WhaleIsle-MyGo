package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) themeFontEditor(visible bool) *theme.Editor {
	s.appearanceVisible = visible
	if s.appearanceEditor == nil && s.cfgStore != nil {
		p, err := s.cfgStore.Appearance()
		s.appearanceEditor = theme.NewEditor(p, s.cfgStore.SaveAppearance)
		if err != nil {
			s.appearanceEditor.Status = err.Error()
		}
	}
	return s.appearanceEditor
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

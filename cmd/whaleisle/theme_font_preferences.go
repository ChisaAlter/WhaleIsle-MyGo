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
	p := theme.CurrentPreferences()
	colors := theme.Resolve(id).Active(mygo.Theme.IsDark())
	t := *c.Theme()
	t.Font = p.Interface.Family
	t.FontSize = p.Interface.Size
	t.Background = chrome.Color(colors.BG)
	t.Text = chrome.Color(colors.FG)
	t.TextMuted = chrome.Color(colors.Muted)
	t.Surface = chrome.Color(colors.Field)
	t.SurfaceHover = chrome.Color(colors.Hover)
	t.Border = chrome.Color(colors.Border)
	t.Accent = chrome.Color(colors.Accent)
	t.AccentText = chrome.Color(colors.ButtonFG)
	t.Focus = t.Accent
	t.Selection = t.Accent.Alpha(.3)
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

package main

import (
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"whaleisle/internal/ui/chrome"
)

func (s *mainState) configurePointerEffects(c *ui.Context) {
	g := s.wallpaperEditor()
	if g == nil {
		return
	}
	p := g.Prefs
	config := ui.DecorativeEffects{PointerEnabled: p.CursorEnabled, PointerEffect: p.CursorEffect, Speed: float32(p.CursorSpeed), Size: float32(p.CursorSize), MetallicButtons: p.MetallicButtons}
	for _, raw := range p.CursorColors {
		raw = strings.TrimSpace(raw)
		if len(raw) != 7 || !strings.HasPrefix(raw, "#") {
			continue
		}
		if _, err := strconv.ParseUint(raw[1:], 16, 24); err != nil {
			continue
		}
		config.Colors[config.ColorCount] = chrome.Color(raw)
		config.ColorCount++
	}
	ui.SetDecorativeEffects(c, config)
}

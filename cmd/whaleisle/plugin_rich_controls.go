package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

func (s *mainState) pluginRichOwners() []*pluginui.Owner {
	s.mu.Lock()
	defer s.mu.Unlock()
	owners := []*pluginui.Owner{s.nativePlugins.contributions}
	if p := s.nativePlugins.session; p != nil {
		owners = append(owners, p.owner)
	}
	if p := s.nativePlugins.navigation; p != nil {
		owners = append(owners, p.owner)
	}
	return owners
}
func (s *mainState) beginPluginRichFrame(visible bool) {
	for _, owner := range s.pluginRichOwners() {
		owner.BeginNativeFrame(visible)
	}
}

// The final-placement overlay only paints; it covers the window, so it must not take the pointer.
func (s *mainState) endPluginRichFrame(c *ui.Context) {
	ui.Overlay(c, func() {
		ui.Box(c.Key("plugin-rich-final-placement")).Fill().PassThrough().DrawOver(func(_ *ui.Painter, _ ui.Rect) {
			for _, owner := range s.pluginRichOwners() {
				owner.EndNativeFrame()
			}
		})
	})
}

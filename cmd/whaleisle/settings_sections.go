package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// The desktop market is a settings section in the original registry. Keep its
// existing store and installer callbacks, but paint it inside the modal shell.
func (s *mainState) settingsSectionModel(m settings.Model, marketContent func(*ui.Context)) settings.Model {
	s.mu.Lock()
	section := s.settingsEmbeddedSection
	s.mu.Unlock()
	if section == settings.SectionMarket && marketContent != nil {
		m.ActiveSection, m.SectionContent = section, marketContent
	}
	clear := func() {
		s.mu.Lock()
		s.settingsEmbeddedSection = ""
		s.mu.Unlock()
	}
	for _, callback := range []*func(){
		&m.OnSelectDesktop, &m.OnSelectAccount, &m.OnSelectCredentials,
		&m.OnSelectAbout, &m.OnSelectAppearance, &m.OnSelectInterface,
		&m.OnSelectModels, &m.OnSelectMCP, &m.OnSelectPlugins,
		&m.OnSelectSkills, &m.OnSelectAgentPresets, &m.OnOpenRemote,
		&m.OnOpenUsage, &m.OnClose,
	} {
		if fn := *callback; fn != nil {
			*callback = func() { clear(); fn() }
		}
	}
	if fn := m.OnSelectNamespace; fn != nil {
		m.OnSelectNamespace = func(ns string) { clear(); fn(ns) }
	}
	return m
}

package main

import (
	"context"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// usagePageRefreshed fires the usage read once per process: entering the section
// should not reissue the whole stats query on every repaint.
var usagePageRefreshed atomic.Bool

// The desktop market is a settings section in the original registry. Keep its
// existing store and installer callbacks, but paint it inside the modal shell.
func (s *mainState) settingsSectionModel(m settings.Model, sectionContent func(*ui.Context, settings.SectionID)) settings.Model {
	s.mu.Lock()
	section := s.settingsEmbeddedSection
	s.mu.Unlock()
	if section == settings.SectionUsageStats && s.usage != nil && usagePageRefreshed.CompareAndSwap(false, true) {
		s.usage.RefreshStats(context.Background(), true)
	}
	if (section == settings.SectionMarket || section == settings.SectionRemote || section == settings.SectionRemoteWorkspace || section == settings.SectionUsageStats) && sectionContent != nil {
		// Capture the section: OnClose clears settingsEmbeddedSection before the
		// modal builds its content, and a live read would paint the chat behind
		// the dialog with the same ListState as the page (panic: ListState shown
		// by two Lists).
		embedded := section
		m.ActiveSection = embedded
		m.SectionContent = func(c *ui.Context) { sectionContent(c, embedded) }
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

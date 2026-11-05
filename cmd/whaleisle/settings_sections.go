package main

import (
	"context"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// Read on page entry and Host replacement, not on every immediate-mode repaint.
var usagePageReads sync.Map // *mainState -> current Host/store identity

// The desktop market is a settings section in the original registry. Keep its
// existing store and installer callbacks, but paint it inside the modal shell.
func (s *mainState) settingsSectionModel(m settings.Model, sectionContent func(*ui.Context, settings.SectionID)) settings.Model {
	s.mu.Lock()
	section := s.settingsEmbeddedSection
	s.mu.Unlock()
	if section == settings.SectionUsageStats && s.usage != nil {
		owner := s.usage.StatisticsOwner()
		previous, loaded := usagePageReads.LoadOrStore(s, owner)
		if !loaded || previous != owner {
			usagePageReads.Store(s, owner)
			s.usage.RefreshStats(context.Background(), false)
		}
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
		s.leaveSettingsSection(m, section)
	}
	for _, callback := range []*func(){
		&m.OnSelectDesktop, &m.OnSelectAccount, &m.OnSelectCredentials,
		&m.OnSelectAbout, &m.OnSelectAppearance, &m.OnSelectInterface,
		&m.OnSelectMCP, &m.OnSelectPlugins,
		&m.OnSelectSkills, &m.OnSelectAgentPresets, &m.OnOpenRemote, &m.OnOpenRemoteWorkspace,
		&m.OnSelectConversation, &m.OnSelectFeatures, &m.OnOpenMarket,
		&m.OnClose,
	} {
		if fn := *callback; fn != nil {
			*callback = func() { clear(); fn() }
		}
	}
	// Selecting the already mounted page does not unmount its editor. Only a
	// real page transition (or close) retires the card's drafts and read owner.
	if fn := m.OnSelectModels; fn != nil {
		m.OnSelectModels = func() {
			if section != "" || !m.ModelsActive {
				clear()
			}
			fn()
		}
	}
	if fn := m.OnOpenUsage; fn != nil {
		m.OnOpenUsage = func() {
			if section != settings.SectionUsageStats {
				clear()
			}
			fn()
		}
	}
	if fn := m.OnSelectNamespace; fn != nil {
		m.OnSelectNamespace = func(ns string) { clear(); fn(ns) }
	}
	return m
}

// Native and contributed sections share one page-retirement boundary. In
// particular, visiting a plugin section must retire the model card drafts too.
func (s *mainState) leaveSettingsSection(m settings.Model, section settings.SectionID) {
	usagePageReads.Delete(s)
	if section == settings.SectionUsageStats && s.usage != nil {
		s.usage.ClearStatsDateRange()
		s.usage.SetStatsDays(7)
	}
	if m.Store != nil {
		m.Store.LeaveServedForms()
		m.Store.LeaveProviderForms()
	}
	credential := s.settingsReferencedCredentials.Snapshot()
	if !credential.Busy {
		s.settingsReferencedCredentials.Edit(credential.Ticket, "")
		s.settingsReferencedCredentials.StageClear(credential.Ticket, false)
	}
	s.mu.Lock()
	s.settingsEmbeddedSection = ""
	s.settingsConversationActive = false
	s.settingsFeaturesActive = false
	s.mu.Unlock()
}

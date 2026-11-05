package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/providerdiscovery"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) runProviderDiscovery(action func(context.Context)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		action(ctx)
		s.updateProviderProfilesUI()
	}()
}
func (s *mainState) refreshProviderDiscovery() {
	s.runProviderDiscovery(s.settingsProviderDiscovery.Refresh)
}
func (s *mainState) providerDiscoveryAccess() *settings.ProviderDiscoveryAccess {
	return &settings.ProviderDiscoveryAccess{
		Snapshot: s.settingsProviderDiscovery.Snapshot(), OnRefresh: s.refreshProviderDiscovery,
		OnProbeEditor: func(target providerdiscovery.EditorProbe, complete func([]providerdiscovery.Candidate, error)) func() {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			go func() {
				defer cancel()
				candidates, err := s.settingsProviderDiscovery.ProbeEditor(ctx, target)
				s.mu.Lock()
				win := s.win
				s.mu.Unlock()
				if win != nil {
					win.Update(func() { complete(candidates, err) })
				}
				s.updateProviderProfilesUI()
			}()
			return cancel
		},
		OnSelect: func(ticket uint64, id string) {
			s.settingsProviderDiscovery.Select(ticket, id)
			s.updateProviderProfilesUI()
		},
		OnEdit: func(ticket uint64, field, value string) {
			s.settingsProviderDiscovery.Edit(ticket, field, value)
			s.updateProviderProfilesUI()
		},
		OnDiscover: func(ticket uint64) {
			s.runProviderDiscovery(func(ctx context.Context) { s.settingsProviderDiscovery.Discover(ctx, ticket) })
		},
		OnPick: func(ticket uint64, id string, picked bool) {
			s.settingsProviderDiscovery.Pick(ticket, id, picked)
			s.updateProviderProfilesUI()
		},
		OnAdopt:   func(ticket uint64) { s.settingsProviderDiscovery.Adopt(ticket); s.updateProviderProfilesUI() },
		OnDiscard: func(ticket uint64) { s.settingsProviderDiscovery.Discard(ticket); s.updateProviderProfilesUI() },
		OnReview: func(ticket uint64, review bool) {
			s.settingsProviderDiscovery.Review(ticket, review)
			s.updateProviderProfilesUI()
		},
		OnSave: func(ticket uint64) {
			s.runProviderDiscovery(func(ctx context.Context) {
				if s.settingsProviderDiscovery.Save(ctx, ticket) {
					s.mu.Lock()
					sessions, set := s.sessions, s.settings
					s.mu.Unlock()
					if sessions != nil {
						sessions.RefreshModelCatalog(context.Background())
					}
					if set != nil {
						set.RefreshDescribe(context.Background())
					}
				}
			})
		},
	}
}

// Check the latest navigation flags after rendering: a tab/close click in that
// same frame must retire the old probe and confirmation callbacks immediately.
func (s *mainState) observeProviderDiscovery(rendered bool) {
	s.mu.Lock()
	visible := rendered && s.settingsOpen && s.settingsModelsActive && !s.quitting
	win := s.win
	s.mu.Unlock()
	if win == nil || !win.IsVisible() || win.IsMinimized() {
		visible = false
	}
	s.settingsProviderDiscovery.ObserveVisible(visible)
}

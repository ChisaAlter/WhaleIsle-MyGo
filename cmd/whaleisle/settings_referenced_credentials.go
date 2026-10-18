package main

import (
	"context"
	"time"
	"whaleisle/internal/ui/settings"
)

func (s *mainState) runReferencedCredentials(action func(context.Context)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		action(ctx)
		s.updateProviderProfilesUI()
	}()
}
func (s *mainState) refreshReferencedCredentials() {
	s.runReferencedCredentials(s.settingsReferencedCredentials.Refresh)
}
func (s *mainState) referencedCredentialsAccess() *settings.ReferencedCredentialsAccess {
	return &settings.ReferencedCredentialsAccess{
		Snapshot:  s.settingsReferencedCredentials.Snapshot(),
		OnRefresh: s.refreshReferencedCredentials,
		OnSelect:  func(id string) { s.settingsReferencedCredentials.Select(id); s.updateProviderProfilesUI() },
		OnEdit:    func(ticket uint64, value string) { s.settingsReferencedCredentials.Edit(ticket, value) },
		OnClear: func(ticket uint64, clear bool) {
			s.settingsReferencedCredentials.StageClear(ticket, clear)
			s.updateProviderProfilesUI()
		},
		OnSave: func(ticket uint64) {
			s.runReferencedCredentials(func(ctx context.Context) { s.settingsReferencedCredentials.Save(ctx, ticket) })
		},
	}
}

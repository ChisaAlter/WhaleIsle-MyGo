package main

import (
	"context"
	"time"
	"whaleisle/internal/dsh/api"
	settingsstore "whaleisle/internal/store/settings"
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
		OnSaveProvider: func(ticket uint64, binding *settingsstore.NamespaceBinding) {
			s.runReferencedCredentials(func(ctx context.Context) {
				saved := false
				s.settingsReferencedCredentials.SaveWithSettings(ctx, ticket, func(ctx context.Context) (api.SettingsNamespaceView, error) {
					written, err := binding.SaveProviderSettings(ctx)
					saved = err == nil
					return written, err
				})
				if saved {
					binding.RefreshProviderSettings(context.Background())
				}
			})
		},
		OnCreateProvider: func(binding *settingsstore.NamespaceBinding, provider, key string, complete func(bool, bool)) {
			s.runReferencedCredentials(func(ctx context.Context) {
				settingsSaved := false
				accepted, saved := s.settingsReferencedCredentials.CreateWithSettings(ctx, provider, key, func(ctx context.Context) (api.SettingsNamespaceView, error) {
					written, err := binding.SaveProviderSettings(ctx)
					settingsSaved = err == nil
					if err == nil {
						binding.SetForm("@provider-create-committed", provider)
					}
					return written, err
				})
				if settingsSaved {
					binding.RefreshProviderSettings(context.Background())
				}
				s.mu.Lock()
				win := s.win
				s.mu.Unlock()
				if win != nil {
					win.Update(func() { complete(accepted, saved) })
				}
			})
		},
	}
}

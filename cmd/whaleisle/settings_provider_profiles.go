package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/ui/settings"
)

type notifyingProviderCaller struct {
	providerprofiles.Caller
	notify func()
}

func (c notifyingProviderCaller) Call(ctx context.Context, method string, args, out any) error {
	c.notify()
	return c.Caller.Call(ctx, method, args, out)
}
func (s *mainState) runProviderProfiles(action func(context.Context, providerprofiles.Caller)) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	var caller providerprofiles.Caller
	if client != nil {
		caller = notifyingProviderCaller{Caller: client, notify: s.updateProviderProfilesUI}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		action(ctx, caller)
		s.updateProviderProfilesUI()
	}()
}
func (s *mainState) refreshProviderProfiles() {
	s.runProviderProfiles(func(ctx context.Context, c providerprofiles.Caller) { s.settingsProviderProfiles.Refresh(ctx, c) })
}
func (s *mainState) providerProfilesAccess() *settings.ProviderProfilesAccess {
	snap := s.settingsProviderProfiles.Snapshot()
	id := ""
	if snap.Pending != nil {
		id = snap.Pending.ID
	}
	return &settings.ProviderProfilesAccess{Snapshot: snap, OnRefresh: s.refreshProviderProfiles,
		OnDelete: func(id string) { s.settingsProviderProfiles.RequestDelete(id); s.updateProviderProfilesUI() },
		OnCancel: func() { s.settingsProviderProfiles.CancelDelete(); s.updateProviderProfilesUI() },
		OnConfirm: func() {
			s.runProviderProfiles(func(ctx context.Context, c providerprofiles.Caller) {
				s.settingsProviderProfiles.DeleteExpected(ctx, c, id)
			})
		},
	}
}

func (s *mainState) updateProviderProfilesUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

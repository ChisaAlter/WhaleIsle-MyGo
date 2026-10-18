package main

import (
	"context"
	"errors"
	"time"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/ui/settings"
)

type notifyingProviderCaller struct {
	providerprofiles.Caller
	notify  func()
	current func() bool
}

func (c notifyingProviderCaller) Call(ctx context.Context, method string, args, out any) error {
	if !c.current() {
		return errors.New("Host changed; refresh provider profiles before continuing")
	}
	c.notify()
	return c.Caller.Call(ctx, method, args, out)
}
func (s *mainState) runProviderProfiles(action func(context.Context, providerprofiles.Caller)) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	var caller providerprofiles.Caller
	if client != nil {
		caller = notifyingProviderCaller{Caller: client, notify: s.updateProviderProfilesUI, current: func() bool { s.dshMu.Lock(); defer s.dshMu.Unlock(); return s.dshClient == client }}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		action(ctx, caller)
		s.updateProviderProfilesUI()
	}()
}
func (s *mainState) refreshProviderProfiles() {
	epoch := s.settingsProviderProfiles.Epoch()
	s.runProviderProfiles(func(ctx context.Context, c providerprofiles.Caller) {
		s.settingsProviderProfiles.RefreshExpected(ctx, c, epoch)
	})
}
func (s *mainState) providerProfilesAccess() *settings.ProviderProfilesAccess {
	snap := s.settingsProviderProfiles.Snapshot()
	id := ""
	if snap.Pending != nil {
		id = snap.Pending.ID
	}
	return &settings.ProviderProfilesAccess{Snapshot: snap, OnRefresh: s.refreshProviderProfiles,
		OnDelete: func(id string) {
			s.settingsProviderProfiles.RequestDeleteAtEpoch(id, snap.Epoch)
			s.updateProviderProfilesUI()
		},
		OnCancel: func() { s.settingsProviderProfiles.CancelDelete(); s.updateProviderProfilesUI() },
		OnConfirm: func() {
			s.runProviderProfiles(func(ctx context.Context, c providerprofiles.Caller) {
				s.settingsProviderProfiles.DeleteAtEpoch(ctx, c, id, snap.Epoch)
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

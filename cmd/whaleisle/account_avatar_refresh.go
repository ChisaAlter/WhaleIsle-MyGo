package main

import (
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/sidebar"
)

func (s *mainState) retryAccountAvatar() {
	s.mu.Lock()
	store, win := s.usage, s.win
	s.mu.Unlock()
	if store == nil {
		return
	}
	v := store.Snapshot()
	if v.Details.Profile.Value.AvatarURL == nil {
		return
	}
	raw := *v.Details.Profile.Value.AvatarURL
	invalidate := func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	settings.RetryAccountAvatar(raw, invalidate)
	sidebar.RetryAccountAvatar(raw, invalidate)
}

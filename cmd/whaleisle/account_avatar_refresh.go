package main

import (
	"whaleisle/internal/ui/settings"
)

func (s *mainState) retryAccountAvatar() {
	s.mu.Lock()
	store, win := s.usage, s.win
	s.mu.Unlock()
	if store == nil {
		return
	}
	v := store.Snapshot().Value
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
}

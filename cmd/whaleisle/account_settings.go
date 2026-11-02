package main

import (
	"context"

	"whaleisle/internal/desktop/deeplink"
)

// openAccountSettings is shared by the sidebar menu and settings navigation.
// Opening the account page must not start authentication or open an editor.
func (s *mainState) openAccountSettings() {
	s.selectMainPanel("")
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionSettings, Section: "account"})
	s.mu.Lock()
	store, usage, sessions := s.settings, s.usage, s.sessions
	s.mu.Unlock()
	if store != nil {
		store.RefreshDescribe(context.Background())
	}
	if usage != nil {
		id := ""
		if sessions != nil {
			id = sessions.SelectedSessionID()
		}
		usage.Refresh(context.Background(), id)
	}
}

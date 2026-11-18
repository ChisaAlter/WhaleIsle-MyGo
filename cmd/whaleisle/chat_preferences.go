package main

import (
	"whaleisle/internal/desktop/webview"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/settings"
)

func (s *mainState) openChatLink(rawURL string) {
	p := settings.ChatPreferencesFrom(s.settings.Snapshot())
	if p.LinkOpening == "new-tab" {
		if err := webview.OpenExternal(rawURL); err != nil && s.sessions != nil {
			s.sessions.SetLastAction(errcopy.Sentence(err))
		}
		return
	}
	s.showBrowserPreview("")
	s.browserController.NewTab(rawURL)
}

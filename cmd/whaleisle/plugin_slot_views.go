package main

import "whaleisle/internal/harness"

func (s *mainState) openInstalledPluginView(name string) {
	s.mu.Lock()
	st := s.bootStatus
	if st.State != harness.Ready {
		s.mu.Unlock()
		return
	}
	s.settingsOpen = false
	s.browserPreviewOpen = false
	s.assistantOpen = false
	s.webviewOpen = true
	s.mu.Unlock()
	s.openWebViewHostPlugin(st.Endpoint.Origin, name)
}

// Revokes the temporary browser transport as soon as its Host generation ends.
func (s *mainState) syncPluginViewHost(st harness.Status) {
	s.mu.Lock()
	host := s.webviewHost
	s.mu.Unlock()
	if host != nil && host.HasSession() && (st.State != harness.Ready || !host.MatchesSession(st.Endpoint.Origin, st.Endpoint.Cookie)) {
		s.closeWebViewHost()
	}
}

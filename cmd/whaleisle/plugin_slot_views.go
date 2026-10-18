package main

import (
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
)

func (s *mainState) openInstalledPluginView(name string) {
	s.mu.Lock()
	st := s.bootStatus
	if st.State != harness.Ready {
		s.mu.Unlock()
		return
	}
	s.settingsOpen = false
	s.browserPreviewOpen = false
	s.assistantConversationGeneration++
	s.assistantOpen = false
	s.webviewOpen = true
	s.mu.Unlock()
	s.openWebViewHostPlugin(st.Endpoint.Origin, name)
}

// Revokes the temporary browser transport as soon as its Host generation ends.
func (s *mainState) syncPluginViewHost(st harness.Status) {
	s.mu.Lock()
	host, sup := s.webviewHost, s.sup
	hostGeneration := s.pluginViewHostGeneration
	s.mu.Unlock()
	if sup != nil {
		actual := sup.Status()
		if actual.Generation != st.Generation || actual.State != st.State {
			return
		}
	}
	if host != nil && host.HasSession() && (st.State != harness.Ready || hostGeneration != st.Generation || !host.MatchesSession(st.Endpoint.Origin, st.Endpoint.Cookie)) {
		s.closeWebViewHost()
		s.mu.Lock()
		s.pluginViewResumeAfterRestart = s.webviewOpen && !s.browserPreviewOpen
		s.mu.Unlock()
	}
	if st.State != harness.Ready {
		return
	}
	s.mu.Lock()
	resume := s.pluginViewResumeAfterRestart && s.webviewOpen && !s.browserPreviewOpen
	s.pluginViewResumeAfterRestart = false
	raw := desktopwebview.RestorePanelURL(st.Endpoint.Origin, s.lastWebViewOrigin, s.lastWebViewURL)
	if resume {
		s.lastWebViewOrigin = st.Endpoint.Origin
	}
	s.mu.Unlock()
	if resume {
		s.openWebViewURL(raw)
	}
}

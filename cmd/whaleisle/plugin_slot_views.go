package main

import (
	"strings"

	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/sidebar"
)

func (s *mainState) openInstalledPluginView(name string) {
	s.openInstalledPluginViewWithLabel(name, "")
}

func (s *mainState) openInstalledPluginViewWithLabel(name, label string) {
	s.mu.Lock()
	st := s.bootStatus
	if st.State != harness.Ready {
		s.mu.Unlock()
		return
	}
	s.settingsOpen = false
	s.browserPreviewOpen = false
	s.webviewOpen = true
	s.activePluginViewLabel = strings.TrimSpace(label)
	s.mu.Unlock()
	s.assistantService().SetOpen(false)
	s.openWebViewHostPlugin(st.Endpoint.Origin, name)
}

func (s *mainState) sidebarPluginEntries(locale string) []sidebar.PluginEntry {
	rows := s.marketCoordinator().SidebarPlugins(locale)
	if len(rows) == 0 {
		return nil
	}
	out := make([]sidebar.PluginEntry, len(rows))
	for i, r := range rows {
		out[i] = sidebar.PluginEntry{ID: r.ID, Label: r.Label, Order: r.Order}
	}
	return out
}

func (s *mainState) sidebarPluginLabel(id string) string {
	for _, item := range s.marketCoordinator().SidebarPlugins(i18n.Locale()) {
		if item.ID == id {
			return item.Label
		}
	}
	return id
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

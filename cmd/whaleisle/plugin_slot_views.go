package main

import (
	"net/url"
	"strings"

	"github.com/egoist/mygo"

	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/sidebar"
)

// Inventory changes revoke an installed page only after a newer authoritative
// read completes. A loading snapshot or static discovery cache cannot revoke it.
func (s *mainState) syncInstalledPluginView() {
	s.mu.Lock()
	host := s.webviewHost
	if s.mainRoute != routeWebView || host == nil || !host.HasSession() {
		s.mu.Unlock()
		return
	}
	raw := s.lastWebViewURL
	observedHost, observedRevision := s.pluginViewInventoryHost, s.pluginViewInventoryRevision
	s.mu.Unlock()
	u, err := url.Parse(raw)
	if err != nil {
		return
	}
	id := u.Query().Get(desktopwebview.DesktopPluginQuery)
	if id == "" {
		return
	}
	snapshot := s.marketCoordinator().InstalledPluginState(id)
	if !snapshot.Known || (snapshot.Owner == observedHost && snapshot.Revision <= observedRevision) {
		return
	}
	s.mu.Lock()
	if s.webviewHost != host || s.lastWebViewURL != raw {
		s.mu.Unlock()
		return
	}
	s.pluginViewInventoryHost, s.pluginViewInventoryRevision = snapshot.Owner, snapshot.Revision
	var retired retiredWebView
	if !snapshot.Enabled {
		retired = s.detachWebViewLocked()
		s.webviewHostErr = resumeText("该插件已禁用或不可用。", "This plugin is disabled or unavailable.")
	}
	s.mu.Unlock()
	retired.close()
}

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
	s.activePluginViewLabel = strings.TrimSpace(label)
	s.mu.Unlock()
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
	win := s.win
	s.mu.Unlock()
	if win != nil {
		mygo.RunOnMain(func() { s.applyPluginViewHost(st) })
		return
	}
	s.applyPluginViewHost(st)
}

func (s *mainState) applyPluginViewHost(st harness.Status) {
	s.mu.Lock()
	if s.bootStatus.Generation != st.Generation || s.bootStatus.State != st.State {
		s.mu.Unlock()
		return
	}
	host, sup := s.webviewHost, s.sup
	hostGeneration := s.pluginViewHostGeneration
	ticket := mainRouteToken{s.mainRoute, s.mainRouteEpoch}
	s.mu.Unlock()
	if sup != nil {
		actual := sup.Status()
		if actual.Generation != st.Generation || actual.State != st.State {
			return
		}
	}
	if host != nil && host.HasSession() && (st.State != harness.Ready || hostGeneration != st.Generation || !host.MatchesSession(st.Endpoint.Origin, st.Endpoint.Cookie)) {
		s.mu.Lock()
		var retired retiredWebView
		if s.webviewHost == host {
			retired = s.detachWebViewLocked()
			s.pluginViewResumeAfterRestart = s.mainRouteCurrentLocked(ticket) && s.mainRoute == routeWebView
		}
		s.mu.Unlock()
		retired.close()
	}
	if st.State != harness.Ready {
		return
	}
	s.mu.Lock()
	resume := s.pluginViewResumeAfterRestart && s.mainRouteCurrentLocked(ticket) && s.mainRoute == routeWebView
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

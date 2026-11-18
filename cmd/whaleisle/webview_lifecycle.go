package main

import (
	"net/url"
	"strings"

	desktopwebview "whaleisle/internal/desktop/webview"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	webviewui "whaleisle/internal/ui/webview"
)

// A surface is detached under the shell lock, then revoked and destroyed
// outside it. Window callbacks must never reenter a held mainState mutex.
type retiredWebView struct {
	host    *desktopwebview.Host
	account *accountPlatformLease
}

func (r retiredWebView) close() {
	closeAccountPlatformLease(r.account)
	if r.host != nil && !r.account.ownsHost(r.host) {
		_ = r.host.Close()
	}
}

func (s *mainState) detachWebViewLocked() retiredWebView {
	host := s.webviewHost
	if host != nil && s.accountReturn.platform == nil {
		s.lastWebViewURL = host.URL()
	}
	retired := retiredWebView{host: host, account: s.detachAccountPlatformLocked()}
	s.webviewHost = nil
	s.settingsOverlayHost = nil
	s.pluginViewResumeAfterRestart = false
	s.webviewHostErr = ""
	s.activePluginViewLabel = ""
	return retired
}

func (s *mainState) closeMainWebView() {
	s.mu.Lock()
	retired := s.detachWebViewLocked()
	s.mu.Unlock()
	retired.close()
}

func (s *mainState) openMainWebViewPlugin(origin, pluginID string) {
	s.marketCoordinator().PrepareInstalledPlugin()
	s.mu.Lock()
	s.setMainRouteLocked(routeWebView)
	s.lastWebViewOrigin = origin
	s.mu.Unlock()
	s.syncMainRouteResources()
	s.openMainWebViewURL(desktopwebview.SlotURL(origin, pluginID, nil))
}

func (s *mainState) showMainBrowser(rawURL string) {
	s.navigateMainRoute(routeBrowser)
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if s.browserController == nil {
		s.browserController = s.newBrowserController(win)
	}
	if strings.TrimSpace(rawURL) != "" {
		s.browserController.Navigate(rawURL)
	}
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) openMainWebViewURL(browse string) {
	s.closeAccountPlatform()
	s.mu.Lock()
	s.accountReturn.embedded = ""
	win, existing := s.win, s.webviewHost
	ep, generation := s.bootStatus.Endpoint, s.bootStatus.Generation
	ticket := mainRouteToken{s.mainRoute, s.mainRouteEpoch}
	label := strings.TrimSpace(s.activePluginViewLabel)
	allowed := s.mainRoute == routeWebView && !s.quitting
	s.mu.Unlock()
	if win == nil || !allowed {
		return
	}
	browse = strings.TrimSpace(browse)
	if browse == "" {
		s.mu.Lock()
		if s.mainRouteCurrentLocked(ticket) {
			s.webviewHostErr = i18n.T(i18n.KeyCmdURLUnavailable)
		}
		s.mu.Unlock()
		return
	}
	sessionOrigin, sessionCookie := "", ""
	if desktopwebview.IsHostURL(browse, ep.Origin) {
		sessionOrigin, sessionCookie = ep.Origin, ep.Cookie
	}
	var inventory manager.InstalledPluginState
	if u, parseErr := url.Parse(browse); parseErr == nil && sessionOrigin != "" {
		if id := u.Query().Get(desktopwebview.DesktopPluginQuery); id != "" {
			inventory = s.marketCoordinator().InstalledPluginState(id)
			if inventory.Known && !inventory.Enabled {
				s.closeMainWebView()
				s.mu.Lock()
				if s.mainRouteCurrentLocked(ticket) {
					s.webviewHostErr = resumeText("该插件已禁用或不可用。", "This plugin is disabled or unavailable.")
				}
				s.mu.Unlock()
				return
			}
		}
	}
	if existing != nil && existing.Alive() && existing.MatchesSession(sessionOrigin, sessionCookie) {
		err := existing.Navigate(browse)
		s.mu.Lock()
		current := s.mainRouteCurrentLocked(ticket) && s.webviewHost == existing
		if current {
			if err != nil {
				s.webviewHostErr = err.Error()
			} else {
				s.pluginViewInventoryHost, s.pluginViewInventoryRevision = inventory.Owner, inventory.Revision
				s.lastWebViewURL = browse
				s.pluginViewResumeAfterRestart = false
				s.webviewHostErr = ""
			}
		}
		s.mu.Unlock()
		if current && err == nil {
			existing.SetVisible(true)
		}
		return
	}
	if existing != nil {
		s.mu.Lock()
		if s.webviewHost == existing {
			s.webviewHost = nil
		}
		s.mu.Unlock()
		_ = existing.Close()
	}
	if label == "" {
		label = i18n.T(i18n.KeyWebViewTitle)
	}
	host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
		Untrusted: true, SessionOrigin: sessionOrigin, SessionCookie: sessionCookie,
		ShortcutPreferences: s.pageShortcutPreferences(), OnShortcutCloseWindow: s.queueProductShortcutClose,
		OnChange: func() { win.Update(func() {}) }, OnFailure: s.onEmbeddedPageFailure,
		Parent: win, URL: browse, Title: label, TopInset: chrome.TitleBarHeight + webviewui.PanelChromeHeight,
		PermissionPolicy: desktopwebview.PermissionPolicy{
			SessionHost: strings.TrimSpace(sessionOrigin) != "", NotificationsAllowed: systemNotificationsSupported,
		}.Decide,
	})
	s.mu.Lock()
	current := s.mainRouteCurrentLocked(ticket) && s.bootStatus.Generation == generation && s.bootStatus.Endpoint == ep
	if current {
		if err != nil {
			s.webviewHostErr = err.Error()
		} else {
			s.webviewHost = host
			s.pluginViewResumeAfterRestart = false
			s.pluginViewHostGeneration = generation
			s.pluginViewInventoryHost, s.pluginViewInventoryRevision = inventory.Owner, inventory.Revision
			s.lastWebViewURL = browse
			s.webviewHostErr = ""
		}
	}
	s.mu.Unlock()
	if !current && host != nil {
		_ = host.Close()
		return
	}
	if current && err != nil {
		go s.recordWebBootFailure(err.Error())
		return
	}
	if host != nil {
		host.SetVisible(true)
	}
}

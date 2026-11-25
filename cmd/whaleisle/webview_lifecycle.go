package main

import (
	"net/url"
	"runtime"
	"strings"
	"whaleisle/internal/errcopy"

	desktopwebview "whaleisle/internal/desktop/webview"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/product"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	webviewui "whaleisle/internal/ui/webview"
)

// webviewTopInset returns the DIP height reserved above an embedded routeWebView
// WebView2 host. On Windows, webviewui.View owns the top 48 DIP caption band
// aligned with WindowsCaptionButtons (no separate 12 DIP strip above it).
func webviewTopInset() int {
	return webviewTopInsetForOS(runtime.GOOS)
}

func webviewTopInsetForOS(goos string) int {
	if goos == "windows" {
		return webviewui.PanelChromeHeight
	}
	return chrome.TitleBarHeight + webviewui.PanelChromeHeight
}

// panelWindowTitle is the caption of a hosted plugin or Harness panel: this
// product's identity, then the panel's name. The page itself belongs to another
// product and its <title> says so, which is why the host pins this title
// (HostOptions.PinTitle) instead of following the page.
func panelWindowTitle(label string) string {
	if strings.TrimSpace(label) == "" {
		label = i18n.T(i18n.KeyWebViewTitle)
	}
	return product.Name + " — " + strings.TrimSpace(label)
}

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
				s.webviewHostErr = errcopy.Sentence(err)
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
	label = panelWindowTitle(label)
	host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
		Untrusted: true, SessionOrigin: sessionOrigin, SessionCookie: sessionCookie,
		ShortcutPreferences: s.pageShortcutPreferences(), OnShortcutCloseWindow: s.queueProductShortcutClose,
		OnChange: func() { win.Update(func() {}) }, OnFailure: s.onEmbeddedPageFailure,
		Parent: win, URL: browse, Title: label, PinTitle: true, TopInset: webviewTopInset(),
		PermissionPolicy: desktopwebview.PermissionPolicy{
			SessionHost: strings.TrimSpace(sessionOrigin) != "", NotificationsAllowed: systemNotificationsSupported,
		}.Decide,
	})
	s.mu.Lock()
	current := s.mainRouteCurrentLocked(ticket) && s.bootStatus.Generation == generation && s.bootStatus.Endpoint == ep
	if current {
		if err != nil {
			s.webviewHostErr = errcopy.Sentence(err)
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
		go s.recordWebBootFailure(err.Error()) // raw-ok: boot-failure record, not copy
		return
	}
	if host != nil {
		host.SetVisible(true)
	}
}

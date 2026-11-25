package main

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"sync"
	"time"

	"whaleisle/internal/desktop/platformview"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/i18n"
)

type accountPlatformLease struct {
	mu      sync.Mutex
	closed  bool
	cancel  context.CancelFunc
	gateway *platformview.Gateway
	host    *desktopwebview.Host
}

func (l *accountPlatformLease) ownsHost(host *desktopwebview.Host) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.host == host
}

func (l *accountPlatformLease) attachGateway(gateway *platformview.Gateway) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		gateway.Close()
		return false
	}
	l.gateway = gateway
	return true
}

func (l *accountPlatformLease) attachHost(host *desktopwebview.Host) bool {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		_ = host.Close()
		return false
	}
	l.host = host
	l.mu.Unlock()
	return true
}

func (s *mainState) detachAccountPlatformLocked() *accountPlatformLease {
	lease := s.accountReturn.platform
	s.accountReturn.platform = nil
	if lease.ownsHost(s.webviewHost) {
		s.webviewHost = nil
		s.lastWebViewURL = ""
	}
	return lease
}

func closeAccountPlatformLease(lease *accountPlatformLease) {
	if lease == nil {
		return
	}
	lease.mu.Lock()
	if lease.closed {
		lease.mu.Unlock()
		return
	}
	lease.closed = true
	cancel, gateway, host := lease.cancel, lease.gateway, lease.host
	lease.mu.Unlock()
	cancel()
	if gateway != nil {
		gateway.Close()
	}
	if host != nil {
		_ = host.Close()
	}
}

func (s *mainState) closeAccountPlatform() {
	s.mu.Lock()
	lease := s.detachAccountPlatformLocked()
	s.mu.Unlock()
	closeAccountPlatformLease(lease)
}

// The account link is only an entry-point signal. The authenticated Host chooses
// the origin; no caller-supplied URL can receive the account session.
func (s *mainState) openAccountPlatform(page, source string) {
	s.closeWebViewHost()
	s.navigateMainRoute(routeWebView)
	ctx, cancel := context.WithCancel(context.Background())
	lease := &accountPlatformLease{cancel: cancel}
	s.mu.Lock()
	win, sup := s.win, s.sup
	generation := s.bootStatus.Generation
	route := mainRouteToken{s.mainRoute, s.mainRouteEpoch}
	s.accountReturn.platform = lease
	s.accountReturn.err = ""
	s.mu.Unlock()
	if win == nil || sup == nil {
		s.closeAccountPlatform()
		return
	}
	go func() {
		session, err := sup.PlatformSession(ctx)
		win.Update(func() {
			s.mu.Lock()
			current := s.accountReturn.platform == lease && s.bootStatus.Generation == generation && s.mainRouteCurrentLocked(route)
			s.mu.Unlock()
			if !current || ctx.Err() != nil {
				return
			}
			if err != nil {
				s.closeAccountPlatform()
				s.reportAccountAction(err)
				return
			}
			_, offset := time.Now().Zone()
			gateway, err := platformview.New(session, platformview.ClientMetadata{Version: buildVersion, Locale: i18n.Locale(), Platform: platformClientOS(), TimezoneOffsetSeconds: offset})
			if err != nil {
				s.closeAccountPlatform()
				s.reportAccountAction(err)
				return
			}
			if !lease.attachGateway(gateway) {
				return
			}
			entry, _ := gateway.Entry(page)
			path := "/usage"
			if page == "top-up" {
				path = "/top_up"
			}
			host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
				OnFailure:             s.onEmbeddedPageFailure,
				ShortcutPreferences:   s.pageShortcutPreferences(),
				OnShortcutCloseWindow: s.queueProductShortcutClose,
				Parent:                win, URL: entry, DisplayURL: session.Origin + path, Title: "Platform", TopInset: webviewTopInset(),
				TransientProfile: true, Untrusted: true, PreloadScript: gateway.Preload(i18n.Locale()), AllowNavigation: gateway.Allows,
				OnExternalWindow: func(raw string) {
					if target, ok := gateway.ExternalURL(raw); ok {
						_ = desktopwebview.OpenExternal(target)
					}
				},
				OnClosed: func() { cancel(); gateway.Close() }, OnChange: func() { win.Update(func() {}) },
			})
			if err != nil {
				s.closeAccountPlatform()
				s.reportAccountAction(errors.New(i18n.T(i18n.KeyCmdPlatformOpenFailed)))
				return
			}
			if !lease.attachHost(host) {
				return
			}
			s.mu.Lock()
			if s.accountReturn.platform != lease || s.bootStatus.Generation != generation || !s.mainRouteCurrentLocked(route) {
				s.mu.Unlock()
				closeAccountPlatformLease(lease)
				return
			}
			s.webviewHost = host
			s.webviewHostErr = ""
			s.lastWebViewURL = ""
			s.accountReturn.embedded = source
			s.accountReturn.embeddedGeneration = generation
			s.mu.Unlock()
			host.SetVisible(true)
			// Revoke the document on sign-out, account replacement or Host restart.
			go func() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						host.SetAccountLocale(i18n.Locale())
						next, e := sup.PlatformSession(ctx)
						if e != nil || !reflect.DeepEqual(next, session) {
							win.Update(func() {
								s.mu.Lock()
								active := s.accountReturn.platform == lease
								s.mu.Unlock()
								if active {
									s.closeAccountPlatform()
									s.reportAccountAction(errors.New(i18n.T(i18n.KeyCmdPlatformAccountChanged)))
								}
							})
							return
						}
					}
				}
			}()
		})
	}()
}

func platformClientOS() string {
	if runtime.GOOS == "windows" {
		return "desktop-win"
	}
	return "desktop-mac"
}

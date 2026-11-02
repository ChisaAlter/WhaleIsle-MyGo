package main

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"time"

	"whaleisle/internal/desktop/platformview"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	webviewui "whaleisle/internal/ui/webview"
)

type accountPlatformLease struct {
	cancel  context.CancelFunc
	gateway *platformview.Gateway
	host    *desktopwebview.Host
}

func (s *mainState) closeAccountPlatform() {
	s.mu.Lock()
	lease := s.accountReturn.platform
	s.accountReturn.platform = nil
	if lease != nil && s.webviewHost == lease.host {
		s.webviewHost = nil
		s.lastWebViewURL = ""
	}
	s.mu.Unlock()
	if lease != nil {
		lease.cancel()
		if lease.gateway != nil {
			lease.gateway.Close()
		}
		if lease.host != nil {
			_ = lease.host.Close()
		}
	}
}

// The account link is only an entry-point signal. The authenticated Host chooses
// the origin; no caller-supplied URL can receive the account session.
func (s *mainState) openAccountPlatform(page, source string) {
	s.closeWebViewHost()
	ctx, cancel := context.WithCancel(context.Background())
	lease := &accountPlatformLease{cancel: cancel}
	s.mu.Lock()
	win, sup := s.win, s.sup
	generation := s.bootStatus.Generation
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
			current := s.accountReturn.platform == lease && s.bootStatus.Generation == generation
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
			lease.gateway = gateway
			entry, _ := gateway.Entry(page)
			path := "/usage"
			if page == "top-up" {
				path = "/top_up"
			}
			host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
				ShortcutPreferences:   s.pageShortcutPreferences(),
				OnShortcutCloseWindow: s.queueProductShortcutClose,
				Parent:                win, URL: entry, DisplayURL: session.Origin + path, Title: "Platform", TopInset: chrome.TitleBarHeight + webviewui.PanelChromeHeight,
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
			lease.host = host
			s.mu.Lock()
			s.webviewHost = host
			s.webviewHostErr = ""
			s.lastWebViewURL = ""
			s.settingsOpen = false
			s.usageOpen = false
			s.webviewOpen = true
			s.assistantOpen = false
			s.assistantConversationGeneration++
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

package mygo

import (
	"context"
	"errors"
)

var ErrBrowserCookiesUnsupported = errors.New("mygo: Cookie clearing requires a ready, verified Browser-only profile")

type browserCookieToolsBackend interface {
	BrowserCookieToolsAvailable() bool
	ClearBrowserLoginCookies(func(error))
}

// BrowserCookieToolsAvailable reports whether the current page has a verified
// Browser-only persistent profile. Ordinary shared-profile pages are rejected.
func (p *Page) BrowserCookieToolsAvailable() bool {
	return onMainValue(func() bool {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil || !p.w.disableBindings {
			return false
		}
		n, ok := p.w.native.(browserCookieToolsBackend)
		return ok && n.BrowserCookieToolsAvailable()
	})
}

// ClearBrowserLoginCookies clears ALL cookies, including login cookies, in the
// verified BrowserProfile shared by Browser tabs. This can sign users out of
// every Browser tab. Local storage, IndexedDB, WebSQL, service workers and HTTP
// cache are retained; other profiles are untouched. Call only after the user
// explicitly confirms this scope. No arbitrary CDP or native bridge is exposed.
func (p *Page) ClearBrowserLoginCookies(ctx context.Context) error {
	return p.browserToolCall(ctx, func(n browserToolsBackend, done func(error)) {
		cookies, ok := n.(browserCookieToolsBackend)
		if !ok || !cookies.BrowserCookieToolsAvailable() {
			done(ErrBrowserCookiesUnsupported)
			return
		}
		cookies.ClearBrowserLoginCookies(done)
	})
}

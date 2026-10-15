package mygo

import (
	"context"
	"errors"
)

var ErrBrowserSiteDataUnsupported = errors.New("mygo: site-data clearing requires WebView2 Profile2 and a ready, verified Browser workspace profile")

type browserSiteDataBackend interface {
	BrowserSiteDataToolsAvailable() bool
	ClearBrowserSiteData(func(error))
}

// BrowserSiteDataToolsAvailable reports whether this page can request the exact
// five-kind cleanup below. Backends without the optional capability return false.
func (p *Page) BrowserSiteDataToolsAvailable() bool {
	return onMainValue(func() bool {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil || !p.w.disableBindings {
			return false
		}
		n, ok := p.w.native.(browserSiteDataBackend)
		return ok && n.BrowserSiteDataToolsAvailable()
	})
}

// ClearBrowserSiteData clears all Cookies, localStorage, IndexedDB, WebSQL and
// Service Workers in this page's verified BrowserPartition, across all origins
// and tabs in that partition. It does not clear HTTP cache, CacheStorage, file
// systems, history, passwords, settings, other partitions or the default profile.
// The caller must first obtain explicit user confirmation of this destructive
// scope. Cancellation stops waiting, not an already submitted native operation;
// never automatically retry an uncertain result. No arbitrary flags are exposed.
func (p *Page) ClearBrowserSiteData(ctx context.Context) error {
	return p.browserToolCall(ctx, func(n browserToolsBackend, done func(error)) {
		site, ok := n.(browserSiteDataBackend)
		if !ok || !site.BrowserSiteDataToolsAvailable() {
			done(ErrBrowserSiteDataUnsupported)
			return
		}
		site.ClearBrowserSiteData(done)
	})
}

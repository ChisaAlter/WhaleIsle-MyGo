package mygo

import (
	"context"
	"errors"
)

// BrowserColorScheme overrides the page's prefers-color-scheme media feature.
// It does not inject CSS or enable scripts, bindings, permissions or DevTools UI.
type BrowserColorScheme string

const (
	BrowserColorSystem BrowserColorScheme = "system"
	BrowserColorLight  BrowserColorScheme = "light"
	BrowserColorDark   BrowserColorScheme = "dark"
)

var ErrBrowserToolsUnsupported = errors.New("mygo: browser appearance and cache tools are unavailable on this backend")

type browserToolsBackend interface {
	BrowserToolsAvailable() bool
	SetBrowserColorScheme(string, func(error))
	ClearBrowserHTTPCache(func(error))
}

// BrowserToolsAvailable reports support for the narrow native controls below.
// Backends without the optional capability fail closed without interface churn.
func (p *Page) BrowserToolsAvailable() bool {
	return onMainValue(func() bool {
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil || !p.w.disableBindings {
			return false
		}
		n, ok := p.w.native.(browserToolsBackend)
		return ok && n.BrowserToolsAvailable()
	})
}

func (p *Page) browserToolCall(ctx context.Context, call func(browserToolsBackend, func(error))) error {
	if ctx == nil {
		return errors.New("mygo: browser tool context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A cancellation and one native completion may both arrive. Neither blocks,
	// including a late completion after navigation or destruction.
	done := make(chan error, 2)
	stop := context.AfterFunc(ctx, func() { deliver(done, ctx.Err()) })
	defer stop()
	ran := false
	onMain(func() {
		ran = true
		if err := ctx.Err(); err != nil {
			deliver(done, err)
			return
		}
		if p == nil || p.w == nil || p.w.native == nil || p.w.content != nil {
			deliver(done, errDestroyed)
			return
		}
		n, ok := p.w.native.(browserToolsBackend)
		if !p.w.disableBindings || !ok || !n.BrowserToolsAvailable() {
			deliver(done, ErrBrowserToolsUnsupported)
			return
		}
		call(n, func(err error) { deliver(done, err) })
	})
	if !ran {
		return errDestroyed
	}
	err := await(done)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// SetBrowserColorScheme is host initiated and accepts only three fixed values.
// There is deliberately no public arbitrary DevTools-protocol bridge.
func (p *Page) SetBrowserColorScheme(ctx context.Context, scheme BrowserColorScheme) error {
	switch scheme {
	case BrowserColorSystem, BrowserColorLight, BrowserColorDark:
	default:
		return errors.New("mygo: invalid browser color scheme")
	}
	return p.browserToolCall(ctx, func(n browserToolsBackend, done func(error)) { n.SetBrowserColorScheme(string(scheme), done) })
}

// ClearBrowserHTTPCache clears HTTP cache for this page's entire WebView profile,
// including every page sharing it, not just this URL or tab. A BrowserProfile
// page uses its specific BrowserPartition. Explain the scope and confirm first.
// Cookies, local storage and profile directories are not deleted by this API.
func (p *Page) ClearBrowserHTTPCache(ctx context.Context) error {
	return p.browserToolCall(ctx, func(n browserToolsBackend, done func(error)) { n.ClearBrowserHTTPCache(done) })
}

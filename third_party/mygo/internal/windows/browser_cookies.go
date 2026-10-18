//go:build windows && (amd64 || arm64)

package windows

import "errors"

// The environment pointer is published only after Browser-only UDF validation
// in browser_environment.go. Do not fall back to the application's shared env.
func (w *window) BrowserCookieToolsAvailable() bool {
	p := w.browser
	if !w.BrowserToolsAvailable() || w.destroying || !w.opts.BrowserProfile || p == nil || p.key != w.opts.BrowserPartition || p != w.b.browsers[w.opts.BrowserPartition] || p.env == 0 || p.err != nil || p.owners.closed {
		return false
	}
	_, owned := p.owners.members[w]
	return owned
}

func (w *window) ClearBrowserLoginCookies(done func(error)) {
	if !w.BrowserCookieToolsAvailable() {
		done(errors.New("mygo: Cookie clearing requires a ready, verified Browser-only profile"))
		return
	}
	// Official protocol, no parameters and no origin enumeration:
	// https://github.com/ChromeDevTools/devtools-protocol/blob/master/pdl/domains/Network.pdl
	// Reuse the existing native CallDevToolsProtocolMethod adapter, not page JS.
	w.devtools("Network.clearBrowserCookies", "{}", func(_ string, err error) { done(err) })
}

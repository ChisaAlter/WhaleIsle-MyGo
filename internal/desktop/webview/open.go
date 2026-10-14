package webview

import (
	"errors"
	"net/url"
	"strings"

	"github.com/egoist/mygo"
)

// OpenExternal opens an http(s) URL in the system default browser via
// mygo.Shell.OpenExternal.
func OpenExternal(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("webview: empty URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return errors.New("webview: only http(s) URLs are allowed")
	}
	return mygo.Shell.OpenExternal(raw)
}

// HarnessBrowseURL returns the harness web UI entry URL for a dsh web
// origin (scheme://host:port from harness.Endpoint.Origin).
func HarnessBrowseURL(origin string) string {
	origin = strings.TrimSpace(origin)
	origin = strings.TrimRight(origin, "/")
	if origin == "" {
		return ""
	}
	return origin + "/"
}

package webview

import (
	"net/url"
	"strings"
)

// RestorePanelURL keeps a Harness panel's path, query and fragment across host
// recreation (including a changed runtime port). Only an address belonging to
// the previously recorded Harness origin can be transplanted to the new one.
// External documents and credential-bearing URLs are never restored as plugins.
func RestorePanelURL(origin, previousOrigin, previousURL string) string {
	parse := func(raw string) *url.URL {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil
		}
		return u
	}
	current := parse(origin)
	if current == nil {
		return ""
	}
	current.Path, current.RawPath, current.RawQuery, current.Fragment = "/", "", "", ""
	current.RawFragment = ""
	current.ForceQuery = false
	fallback := current.String()
	old, saved := parse(previousOrigin), parse(previousURL)
	if old == nil || saved == nil {
		return fallback
	}
	if !strings.EqualFold(old.Scheme, saved.Scheme) || !strings.EqualFold(old.Host, saved.Host) {
		return fallback
	}
	saved.Scheme, saved.Host = current.Scheme, current.Host
	return saved.String()
}

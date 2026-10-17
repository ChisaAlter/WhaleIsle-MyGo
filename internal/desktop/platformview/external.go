package platformview

import (
	"net/url"
	"strings"
)

func (g *Gateway) privateHeaders() []string {
	out := []string{"authorization", "proxy-authorization", "cookie", "x-client-platform", "x-client-version", "x-client-locale", "x-client-bundle-id", "x-client-timezone-offset"}
	for name := range g.session.RequestHeaders {
		out = append(out, strings.ToLower(name))
	}
	return out
}

// ExternalURL implements the original HTTPS popup-only handoff. No browser
// profile, deployment headers, Host binding or account token accompanies it.
func (g *Gateway) ExternalURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Host == "" {
		return "", false
	}
	if u.Scheme+"://"+u.Host == g.origin {
		if strings.HasPrefix(u.Path, "/__whale_platform_") {
			return "", false
		}
		origin, _ := url.Parse(g.session.Origin)
		u.Scheme = origin.Scheme
		u.Host = origin.Host
	}
	if u.Scheme != "https" {
		return "", false
	}
	decoded, err := url.QueryUnescape(u.String())
	if err != nil {
		return "", false
	}
	for _, secret := range []string{g.key, g.session.Token} {
		if secret != "" && (strings.Contains(u.String(), secret) || strings.Contains(decoded, secret)) {
			return "", false
		}
	}
	return u.String(), true
}

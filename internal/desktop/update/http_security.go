package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxUpdateRedirects = 10

func normalizedURLHost(u *url.URL) string {
	if u == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func isPrivateOrSpecialIP(host string) bool {
	h := strings.TrimSuffix(strings.TrimSpace(host), ".")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// validateDownloadURL ensures remote update requests use HTTPS without
// embedded credentials, allowing plain HTTP only on loopback addresses for local tests.
func validateDownloadURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("update URL must be an HTTPS URL without userinfo")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "https" {
		return u, nil
	}
	if scheme == "http" && isLoopbackHost(u.Hostname()) {
		return u, nil
	}
	return nil, fmt.Errorf("update URL must use HTTPS (HTTP is allowed only for loopback tests)")
}

func isTrustedReleaseCDNHost(initialHost, nextHost string) bool {
	switch {
	case initialHost == "github.com" || initialHost == "api.github.com" || strings.HasSuffix(initialHost, ".github.com"):
		return nextHost == "github.com" ||
			strings.HasSuffix(nextHost, ".github.com") ||
			strings.HasSuffix(nextHost, ".githubusercontent.com")
	case initialHost == "cnb.cool" || strings.HasSuffix(initialHost, ".cnb.cool"):
		return nextHost == "cnb.cool" || strings.HasSuffix(nextHost, ".cnb.cool")
	default:
		return false
	}
}

func validateRedirectTarget(initial, prev, next *url.URL) error {
	nextURL, err := validateDownloadURL(next.String())
	if err != nil {
		return fmt.Errorf("refusing unsafe update redirect: %w", err)
	}
	if prev != nil && strings.EqualFold(prev.Scheme, "https") && !strings.EqualFold(nextURL.Scheme, "https") {
		return fmt.Errorf("refusing update redirect downgrade from %s to %s", prev.Scheme, nextURL.Scheme)
	}
	if initial == nil {
		return nil
	}
	if strings.EqualFold(initial.Scheme, "https") && !strings.EqualFold(nextURL.Scheme, "https") {
		return fmt.Errorf("refusing update redirect downgrade from %s to %s", initial.Scheme, nextURL.Scheme)
	}
	firstHost := normalizedURLHost(initial)
	nextHost := normalizedURLHost(nextURL)
	if nextHost == firstHost {
		return nil
	}
	// Never let a public origin redirect onto loopback or private addresses.
	if !isLoopbackHost(firstHost) && isPrivateOrSpecialIP(nextHost) {
		return fmt.Errorf("refusing update redirect to internal host %q", nextHost)
	}
	if isLoopbackHost(firstHost) && isLoopbackHost(nextHost) {
		return fmt.Errorf("refusing cross-host loopback redirect from %q to %q", firstHost, nextHost)
	}
	if isTrustedReleaseCDNHost(firstHost, nextHost) && strings.EqualFold(nextURL.Scheme, "https") {
		return nil
	}
	return fmt.Errorf("refusing cross-host update redirect from %q to %q", firstHost, nextHost)
}

func hardenDownloadClient(base *http.Client, defaultTimeout time.Duration) *http.Client {
	var client http.Client
	if base != nil {
		client = *base
	}
	if client.Timeout <= 0 {
		client.Timeout = defaultTimeout
	}
	if client.Transport == nil {
		client.Transport = updateTransport()
	}
	prevCheck := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxUpdateRedirects {
			return fmt.Errorf("too many update redirects")
		}
		var initial, prev *url.URL
		if len(via) > 0 {
			initial = via[0].URL
			prev = via[len(via)-1].URL
		}
		if err := validateRedirectTarget(initial, prev, req.URL); err != nil {
			return err
		}
		if prev != nil && !strings.EqualFold(normalizedURLHost(prev), normalizedURLHost(req.URL)) {
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
		}
		if prevCheck != nil {
			return prevCheck(req, via)
		}
		return nil
	}
	return &client
}

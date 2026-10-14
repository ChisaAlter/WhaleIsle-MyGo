package platform

import (
	"net/url"
	"strings"
)

// LocalPDFAllowed permits fragments of exactly one HTTP loopback PDF resource.
// It never broadens an isolated preview to an origin or a directory.
func LocalPDFAllowed(expected, target string) bool {
	u, err := url.Parse(expected)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || !strings.HasSuffix(u.Path, "/document.pdf") {
		return false
	}
	actual, err := url.Parse(target)
	if err != nil || actual.User != nil || actual.RawQuery != "" || actual.RawPath != "" {
		return false
	}
	actual.Fragment = ""
	actual.RawFragment = ""
	return actual.String() == expected
}

package platform

import (
	"net/url"
	"strconv"
	"strings"
)

// LocalHTMLAllowed permits only the exact local HTML capability and fragments
// within it. The capability never grants access to its origin or directory.
func LocalHTMLAllowed(expected, target string) bool {
	u, err := url.Parse(expected)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Opaque != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.String() != expected {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Host != "127.0.0.1:"+strconv.Itoa(port) {
		return false
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) != 3 || parts[0] != "" || len(parts[1]) != 64 || strings.Trim(parts[1], "0123456789abcdefABCDEF") != "" || parts[2] != "document.html" {
		return false
	}
	base, _, _ := strings.Cut(target, "#")
	if base != expected {
		return false
	}
	actual, err := url.Parse(target)
	if err != nil || actual.User != nil || actual.RawQuery != "" || actual.ForceQuery || actual.RawPath != "" {
		return false
	}
	actual.Fragment = ""
	actual.RawFragment = ""
	return actual.String() == expected
}

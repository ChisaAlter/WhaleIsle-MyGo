package platform

import (
	"net/url"
	"strconv"
	"strings"
)

// LocalMediaAllowed grants only a fixed player document. Its sole media
// resource is separately derived, never supplied as a caller-controlled URL.
func LocalMediaAllowed(expected, target string) bool {
	u, err := url.Parse(expected)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Opaque != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.String() != expected {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Host != "127.0.0.1:"+strconv.Itoa(port) {
		return false
	}
	parts := strings.Split(u.Path, "/")
	return len(parts) == 3 && parts[0] == "" && len(parts[1]) == 64 && strings.Trim(parts[1], "0123456789abcdefABCDEF") == "" && parts[2] == "player.html" && target == expected
}
func LocalMediaResourceAllowed(expected, target string) bool {
	if !LocalMediaAllowed(expected, expected) {
		return false
	}
	return target == expected || target == strings.TrimSuffix(expected, "player.html")+"media"
}

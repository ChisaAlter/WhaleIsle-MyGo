//go:build windows

package update

import (
	"net/http"
	"net/url"
	"testing"
)

func TestSystemProxySelectionAndBypass(t *testing.T) {
	for _, tc := range []struct{ target, proxy, bypass, want string }{
		{"https://release.example.test/file", "http=proxy.local:80;https=secure.local:81", "", "http://secure.local:81"},
		{"https://release.example.test/file", "proxy.local:80", "", "http://proxy.local:80"},
		{"https://release.example.test/file", "proxy.local:80", "*.example.test", ""},
		{"http://localhost/file", "proxy.local:80", "<local>", ""},
		{"https://release.example.test/file", "DIRECT", "", ""},
	} {
		u, _ := url.Parse(tc.target)
		got, err := proxyAddress(u, tc.proxy, tc.bypass)
		if err != nil {
			t.Fatal(err)
		}
		value := ""
		if got != nil {
			value = got.String()
		}
		if value != tc.want {
			t.Errorf("%s = %q want %q", tc.target, value, tc.want)
		}
	}
}

func TestUpdaterLoopbackFeedDoesNotReadSystemProxy(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		req, _ := http.NewRequest("GET", "http://"+host+":12345/feed", nil)
		got, err := updateProxy(req)
		if err != nil || got != nil {
			t.Fatalf("loopback %s = %v %v", host, got, err)
		}
	}
}

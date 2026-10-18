//go:build darwin

package darwin

import (
	"strings"
	"testing"
)

func TestBrowserPartitionIdentifier(t *testing.T) {
	a := browserProfileUUID(strings.Repeat("a", 64))
	if a != browserProfileUUID(strings.Repeat("a", 64)) || a == browserProfileUUID(strings.Repeat("b", 64)) {
		t.Fatal("partition UUID must be stable and partition-specific")
	}
	if len(a) != 36 || a[14] != '8' || !strings.ContainsRune("89ab", rune(a[19])) {
		t.Fatalf("invalid nonzero UUIDv8: %q", a)
	}
}
func TestBrowserCapabilitiesRequireLivePartition(t *testing.T) {
	w := &window{}
	if w.BrowserToolsAvailable() || w.BrowserCookieToolsAvailable() || w.BrowserSiteDataToolsAvailable() || w.BrowserColorSchemeAvailable() || w.BrowserDevToolsAvailable() {
		t.Fatal("uninitialized window advertised Browser controls")
	}
	if w.OpenBrowserDevTools() == nil {
		t.Fatal("WKWebView must not claim a public in-app inspector API")
	}
}

package browserpreview

import (
	"strings"
	"testing"
)

func TestBrowserSiteDataConfirmation(t *testing.T) {
	var confirm *browserDataConfirmation[int]
	if takeBrowserDataConfirmation(&confirm, "site-data", 11, 4) {
		t.Fatal("site data must not be cleared without explicit confirmation")
	}
	confirm = &browserDataConfirmation[int]{kind: "site-data", host: 11, epoch: 4}
	for _, mismatch := range []struct {
		kind  string
		host  int
		epoch uint64
	}{{"cookies", 11, 4}, {"cache", 11, 4}, {"all", 11, 4}, {"site-data", 12, 4}, {"site-data", 11, 5}} {
		if takeBrowserDataConfirmation(&confirm, mismatch.kind, mismatch.host, mismatch.epoch) || confirm == nil {
			t.Fatal("confirmation must stay bound to exact cleanup kind, Host and navigation")
		}
	}
	calls := 0 // Synthetic dispatch only: no browser or user data.
	for range 2 {
		if takeBrowserDataConfirmation(&confirm, "site-data", 11, 4) {
			calls++
		}
	}
	if calls != 1 || confirm != nil {
		t.Fatal("a confirmation must permit only one site-data operation")
	}
	for _, scope := range []string{"Cookie", "localStorage", "IndexedDB", "WebSQL", "Service Worker", "HTTP 缓存", "CacheStorage", "其他工作区/插件"} {
		if !strings.Contains(browserSiteDataClearNotice, scope) {
			t.Fatalf("confirmation must disclose %s", scope)
		}
	}
}

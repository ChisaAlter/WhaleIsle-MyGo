package webview

import (
	"net/url"
	"testing"
)

func TestB25DocumentURLRetainsResourceIdentity(t *testing.T) {
	raw, err := DocumentURL("http://127.0.0.1:19200", "session#1", "folder/a #?.md")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(raw)
	if u.Query().Get("desktopDocumentSession") != "session#1" || u.Query().Get("desktopDocumentPath") != "folder/a #?.md" || u.Host != "127.0.0.1:19200" {
		t.Fatal(raw)
	}
	for _, path := range []string{"../escape", "/absolute"} {
		if _, err := DocumentURL("http://127.0.0.1:19200", "s", path); err == nil {
			t.Fatal("accepted outside resource", path)
		}
	}
}

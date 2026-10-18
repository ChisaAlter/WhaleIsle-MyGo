package webview

import "testing"

func TestHarnessBrowseURL(t *testing.T) {
	if got := HarnessBrowseURL("http://127.0.0.1:3080"); got != "http://127.0.0.1:3080/" {
		t.Fatalf("got %q", got)
	}
	if HarnessBrowseURL("  ") != "" {
		t.Fatal("expected empty for blank origin")
	}
}

func TestOpenExternalRejectsNonHTTP(t *testing.T) {
	if err := OpenExternal("file:///etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
}

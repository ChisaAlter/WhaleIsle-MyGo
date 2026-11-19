package webview

import (
	"testing"

	"whaleisle/internal/product"
)

func TestValidateHTTPURL(t *testing.T) {
	if err := validateHTTPURL("http://127.0.0.1/"); err != nil {
		t.Fatal(err)
	}
	if err := validateHTTPURL("file:///tmp"); err == nil {
		t.Fatal("expected reject")
	}
}

func TestURLOrigin(t *testing.T) {
	if got := urlOrigin("http://127.0.0.1:3080/path"); got != "http://127.0.0.1:3080" {
		t.Fatalf("got %q", got)
	}
}

// TestWindowTitleIsNeverEmpty covers the caption a hosted page window carries: a
// caller that names no page still gets this product's name, so the taskbar and
// Alt-Tab list never show a blank entry.
func TestWindowTitleIsNeverEmpty(t *testing.T) {
	for _, title := range []string{"", "   ", "\t"} {
		if got := windowTitle(title); got != product.Name {
			t.Fatalf("windowTitle(%q) = %q, want %q", title, got, product.Name)
		}
	}
	if got := windowTitle("  文档  "); got != "文档" {
		t.Fatalf("windowTitle trimmed to %q", got)
	}
}

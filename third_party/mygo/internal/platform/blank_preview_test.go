package platform

import (
	"strings"
	"testing"
)

func TestBlankEncoderPolicy(t *testing.T) {
	o := WindowOptions{IsolatedPreview: true}
	if !BlankPreviewOptions(&o) {
		t.Fatal("empty isolated helper rejected")
	}
	for _, mutate := range []func(*WindowOptions){func(o *WindowOptions) { o.BrowserProfile = true }, func(o *WindowOptions) { o.Surface = true }, func(o *WindowOptions) { o.Native = 1 }, func(o *WindowOptions) { o.LocalPDFURL = "x" }, func(o *WindowOptions) { o.LocalPDFViewerURL = "x" }, func(o *WindowOptions) { o.LocalHTMLURL = "x" }, func(o *WindowOptions) { o.LocalMediaURL = "x" }} {
		test := o
		mutate(&test)
		if BlankPreviewOptions(&test) {
			t.Fatal("nonblank option accepted")
		}
	}
	for _, url := range []string{"https://example.com", "http://127.0.0.1:8080/x", "file:///tmp/a", "data:text/html,x", "javascript:alert(1)", "about:blank#x"} {
		if BlankPreviewNavigation(url) {
			t.Fatalf("external navigation accepted: %s", url)
		}
	}
	if !BlankPreviewNavigation("about:blank") {
		t.Fatal("blank navigation rejected")
	}
	for _, rule := range []string{"default-src 'none'", "script-src 'none'", "connect-src 'none'", "frame-src 'none'", "form-action 'none'"} {
		if !strings.Contains(BlankPreviewDocument, rule) {
			t.Fatal("missing CSP rule", rule)
		}
	}
}

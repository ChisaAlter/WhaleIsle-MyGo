package platform

import (
	"strings"
	"testing"
)

func TestPDFViewerExactNavigationAndResources(t *testing.T) {
	base := "http://127.0.0.1:4040/" + strings.Repeat("a", 64) + "/"
	viewer := base + "reader.html"
	if !LocalPDFViewerAllowed(viewer, viewer+"#page=2&zoom=125,0,700") {
		t.Fatal("valid state rejected")
	}
	for _, bad := range []string{base + "document.pdf", base + "reader.js", viewer + "?x=1", "file:///test.pdf", "https://evil.test", viewer + "#" + strings.Repeat("x", 1025)} {
		if LocalPDFViewerAllowed(viewer, bad) {
			t.Fatal("navigation allowed", bad)
		}
	}
	for _, name := range []string{"reader.html", "reader.js", "reader.css", "pdf.mjs", "pdf.worker.mjs", "pdf_viewer.mjs", "pdf_viewer.css", "assets.json", "document.pdf"} {
		if !LocalPDFViewerResourceAllowed(viewer, base+name) {
			t.Fatal("bundled resource denied", name)
		}
	}
	for _, bad := range []string{base + "secret", base + "reader.js?x=1", base + "reader.js#x", base + "pdf.sandbox.mjs", "http://127.0.0.1:4041/" + strings.Repeat("a", 64) + "/reader.js", base + "%72eader.js", base + "../reader.js"} {
		if LocalPDFViewerResourceAllowed(viewer, bad) {
			t.Fatal("resource allowed", bad)
		}
	}
	for _, bad := range []string{"http://127.0.0.1:4040/token/reader.html", viewer + "#x", strings.Replace(viewer, "127.0.0.1", "localhost", 1)} {
		if LocalPDFViewerAllowed(bad, bad) {
			t.Fatal("bad capability allowed", bad)
		}
	}
	if LocalPDFAllowed(viewer, viewer) || LocalHTMLAllowed(viewer, viewer) {
		t.Fatal("reader acquired legacy capability")
	}
}

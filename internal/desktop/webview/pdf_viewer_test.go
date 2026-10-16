package webview

import (
	"strings"
	"testing"
)

func TestPDFReaderCapabilityValidation(t *testing.T) {
	base := "http://127.0.0.1:4000/" + strings.Repeat("a", 64) + "/reader.html"
	if !validLocalPDFViewerURL(base) {
		t.Fatal("valid reader rejected")
	}
	for _, bad := range []string{base + "#page=1", base + "?x=1", strings.Replace(base, "127.0.0.1", "localhost", 1), strings.Replace(base, "reader.html", "document.html", 1), "file:///reader.html"} {
		if validLocalPDFViewerURL(bad) {
			t.Fatal("bad reader accepted", bad)
		}
	}
}

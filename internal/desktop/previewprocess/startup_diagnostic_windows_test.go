//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"strings"
	"testing"
)

func TestHelperStartupDiagnosticBoundedBeforeDocumentOnly(t *testing.T) {
	d := &helperStartupDiagnostic{}
	if n, err := d.Write([]byte(strings.Repeat("x", 5000))); n != 5000 || err != nil {
		t.Fatalf("stderr was not drained: n=%d err=%v", n, err)
	}
	if got := d.finish(); len(got) != 4096 {
		t.Fatalf("startup diagnostic length=%d", len(got))
	}
	_, _ = d.Write([]byte("document-time content must not be retained"))
	if got := d.finish(); got != "" {
		t.Fatal("post-handshake output retained")
	}
}

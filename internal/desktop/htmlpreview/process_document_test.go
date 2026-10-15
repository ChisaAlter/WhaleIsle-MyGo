package htmlpreview

import (
	"strings"
	"testing"
)

func TestProcessDocumentShellKeepsOpaqueSandbox(t *testing.T) {
	shell, err := ProcessDocumentShell("<button onclick='this.textContent=42'>Run</button>", 900, "dark")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"frame.setAttribute('sandbox','allow-scripts')",
		"event.source!==frame.contentWindow",
		"event.origin!=='null'",
		"connect-src",
		"worker-src",
	} {
		if !strings.Contains(shell, required) {
			t.Fatalf("missing %q", required)
		}
	}
	if strings.Contains(shell, "allow-same-origin") {
		t.Fatal("opaque origin was relaxed")
	}
	// The ordinary document path remains independently script-free.
	if !strings.Contains(DocumentCSP, "script-src 'none'") || !strings.Contains(DocumentShell("<script>x()</script>"), `sandbox=""`) {
		t.Fatal("static document isolation changed")
	}
}

func TestProcessDocumentShellRejectsInvalidInput(t *testing.T) {
	for _, width := range []int{0, 239, 1601} {
		if _, err := ProcessDocumentShell("x", width, "light"); err == nil {
			t.Fatalf("accepted width %d", width)
		}
	}
	if _, err := ProcessDocumentShell("", 900, "dark"); err == nil {
		t.Fatal("accepted empty HTML")
	}
}

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
		"frame.setAttribute('sandbox','allow-scripts allow-forms')",
		"whale-visual-reply-theme",
		"form-action",
		"event.source!==frame.contentWindow",
		"event.origin!=='null'",
		"connect-src",
		"worker-src",
	} {
		if !strings.Contains(shell, required) {
			t.Fatalf("missing %q", required)
		}
	}
	if strings.Contains(shell, "allow-same-origin") || strings.Contains(shell, "allow-top-navigation") || strings.Contains(shell, "allow-popups") {
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

func TestLiveThemeUpdatesExistingOpaqueFrame(t *testing.T) {
	shell, err := ProcessDocumentShell("<input value='preserve me'>", 728, "light")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shell, "window.__previewTheme=data=>") {
		t.Fatal("missing live update entry")
	}
	for _, appearance := range []string{"dark", "light"} {
		script, err := ProcessThemeScript(appearance)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(script, `"appearance":"`+appearance+`"`) || strings.Contains(script, "srcdoc") || strings.Contains(script, "document.open") {
			t.Fatalf("theme update reloads content: %s", script)
		}
	}
	if _, err := ProcessThemeScript("arbitrary"); err == nil {
		t.Fatal("accepted invalid theme")
	}
}

func TestLiveThemeUsesCurrentApplicationTokens(t *testing.T) {
	script, err := ProcessThemeTokensScript("light", map[string]string{"--accent": "#123abc", "--surface": "#fedcba"})
	if err != nil || !strings.Contains(script, "#123abc") || !strings.Contains(script, "#fedcba") {
		t.Fatalf("custom palette missing: %s %v", script, err)
	}
	if _, err := ProcessThemeTokensScript("light", map[string]string{"arbitrary": "x"}); err == nil {
		t.Fatal("accepted unrelated command field")
	}
}

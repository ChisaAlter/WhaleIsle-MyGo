package pet

import (
	"strings"
	"testing"
)

// TestRestingPageHasNoPermanentChrome pins the native resting look
// (reference/electron/src/renderer/pet-live2d.html:275-303 shows only #pet):
// no permanent renderer label, and the status affordance only appears on hover,
// keyboard focus or right-click.
func TestRestingPageHasNoPermanentChrome(t *testing.T) {
	page, err := assets.ReadFile("assets/pet.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	for _, want := range []string{
		"#pet-status-toggle{position:fixed;top:4px;right:6px;z-index:3;opacity:0;pointer-events:none",
		"body:hover #pet-status-toggle,#pet-status-toggle:focus-visible",
		"#render-notice:empty{display:none}",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("resting styles missing %q", want)
		}
	}
	// The card starts closed; the toggle is not visible at rest.
	if !strings.Contains(html, `id="pet-status-card" hidden`) {
		t.Fatal("status card is visible at rest")
	}
	js := string(statusCardScript)
	for _, want := range []string{"contextmenu", "show(true)", "Escape", "focus()"} {
		if !strings.Contains(js, want) {
			t.Fatalf("status affordance missing %q", want)
		}
	}
	runtime, err := assets.ReadFile("assets/pet-runtime.js")
	if err != nil {
		t.Fatal(err)
	}
	// The rig label must not be written unconditionally any more.
	if strings.Contains(string(runtime), "notice.textContent='原版分层动画 / Layered rig'") {
		t.Fatal("the rig label is still permanent")
	}
	if !strings.Contains(string(runtime), "if(!serverNotice){notice.textContent='';}") {
		t.Fatal("a successful paint does not clear the label")
	}
}

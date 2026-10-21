package browserpreview

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestExternalOpenButtonHandsTheAddressBarURLToTheSystemBrowser pins the real
// address-bar control: 28×28, only with a valid HTTP(S) URL, and the click hands
// exactly that URL to the opener (no draft-only or empty hand-offs).
func TestExternalOpenButtonHandsTheAddressBarURLToTheSystemBrowser(t *testing.T) {
	opened := []string{}
	open := func(url string) error {
		opened = append(opened, url)
		return nil
	}
	view := func(c *ui.Context) {
		ui.Row(c).Height(32).Padding(4).Children(func() {
			externalOpenButton(c, "  https://example.com/dashboard?tab=1  ", open)
		})
	}
	tt := ui.NewTester(view, 640, 120)
	label := browserChromeLabel("在外部打开", "Open externally")
	button, ok := tt.Find(label)
	if !ok {
		t.Fatalf("external button missing: %v", tt.Texts())
	}
	if button.W != 28 || button.H != 28 {
		t.Fatalf("external button = %v×%v, want 28×28", button.W, button.H)
	}
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.com/dashboard?tab=1" {
		t.Fatalf("opened %v", opened)
	}
}

func TestExternalOpenButtonHidesWithoutAValidURL(t *testing.T) {
	view := func(c *ui.Context) {
		ui.Row(c).Height(32).Children(func() {
			for _, draft := range []string{"", "   ", "javascript:alert(1)", "file:///etc/passwd"} {
				externalOpenButton(c, draft, func(string) error { return nil })
			}
		})
	}
	tt := ui.NewTester(view, 640, 120)
	if _, ok := tt.Find(browserChromeLabel("在外部打开", "Open externally")); ok {
		t.Fatalf("external button rendered without a valid URL: %v", tt.Texts())
	}
}

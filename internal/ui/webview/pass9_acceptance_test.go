package webview

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestHostActionsReachTheirCallbacks covers the real chrome strip around the
// WebView2 host: copy, reload, close host and close the panel.
func TestHostActionsReachTheirCallbacks(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })

	copied, reloaded, hostClosed, panelClosed := "", "", 0, 0
	m := Model{
		ThemeID:    "deepseek",
		HostActive: true,
		HostURL:    "http://127.0.0.1:3080/plugin",
		OnCopyURL:  func(u string) { copied = u },
		OnReload:   func(u string) { reloaded = u },
		OnCloseHost: func() {
			hostClosed++
		},
		OnClose: func() { panelClosed++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if err := tt.Click(i18n.T(i18n.KeyWebViewCopyURL)); err != nil {
		t.Fatal(err)
	}
	if copied != m.HostURL {
		t.Fatalf("copied %q", copied)
	}
	if err := tt.Click(i18n.T(i18n.KeyBrowserPreviewReload)); err != nil {
		t.Fatal(err)
	}
	if reloaded != m.HostURL {
		t.Fatalf("reloaded %q", reloaded)
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewCloseHost)); err != nil {
		t.Fatal(err)
	}
	if hostClosed != 1 {
		t.Fatalf("host closes = %d", hostClosed)
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewBack)); err != nil {
		t.Fatal(err)
	}
	if panelClosed != 1 {
		t.Fatalf("panel closes = %d", panelClosed)
	}
}

// TestInactiveHostOffersNoFakeData checks the no-host body: no fake URL or
// connection claim, only the real origin and the two real actions.
func TestInactiveHostOffersNoFakeData(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })

	openedHost, openedBrowser := 0, 0
	m := Model{
		ThemeID:       "deepseek",
		HarnessOrigin: "http://127.0.0.1:3080",
		OnOpenHost:    func() { openedHost++ },
		OnOpenBrowser: func() { openedBrowser++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if !tt.HasText(i18n.T(i18n.KeyWebViewNotOpen)) {
		t.Fatalf("missing not-open state: %v", tt.Texts())
	}
	if tt.HasText("http://127.0.0.1:3080/plugin") {
		t.Fatalf("invented a loaded URL: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewOpenHost)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewOpenBrowser)); err != nil {
		t.Fatal(err)
	}
	// One auto-open on the first frame plus this explicit retry.
	if openedHost != 2 || openedBrowser != 1 {
		t.Fatalf("host=%d browser=%d", openedHost, openedBrowser)
	}
}

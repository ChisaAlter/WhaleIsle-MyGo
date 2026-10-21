package webview

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestWebView2RuntimeErrorOffersADownload(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:   "deepseek-light",
			HostError: "WebView2 runtime is not installed on this machine",
			OnOpenURL: func(u string) { opened = u },
		})
	}, 900, 600)
	if err := tt.Click("Download the WebView2 runtime"); err != nil {
		t.Fatalf("download action missing: %v", err)
	}
	if opened != "https://go.microsoft.com/fwlink/p/?LinkId=2124703" {
		t.Fatalf("download action opened %q", opened)
	}

	// Other errors do not pretend a runtime download would help.
	other := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ThemeID: "deepseek-light", HostError: "navigation failed", OnOpenURL: func(string) {}})
	}, 900, 600)
	if _, ok := other.Find("Download the WebView2 runtime"); ok {
		t.Fatal("runtime download offered for an unrelated error")
	}
}

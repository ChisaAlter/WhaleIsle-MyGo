package webview

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func webviewTestLocale(t *testing.T) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

// TestViewOpensTheHostOnce proves the panel entry loads the real WebView2
// chrome strip instead of stopping on a description of what could be built.
func TestViewOpensTheHostOnce(t *testing.T) {
	webviewTestLocale(t)
	opened, browser := 0, 0
	m := Model{
		HarnessOrigin: "http://127.0.0.1:3080",
		ThemeID:       "deepseek",
		OnOpenHost:    func() { opened++ },
		OnOpenBrowser: func() { browser++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if opened != 1 {
		t.Fatalf("host opens = %d, want 1", opened)
	}
	if browser != 0 {
		t.Fatalf("browser opens = %d, want 0", browser)
	}
	if !tt.HasText(i18n.T(i18n.KeyWebViewNotOpen)) {
		t.Fatalf("missing not-open state: %v", tt.Texts())
	}
	texts := strings.Join(tt.Texts(), "\n")
	for _, banned := range []string{"stub", "无边框", "desktopPlugin", "第三方插件需"} {
		if strings.Contains(texts, banned) {
			t.Fatalf("developer placeholder %q is still rendered:\n%s", banned, texts)
		}
	}
}

// TestViewRetriesTheHostAfterAnError leaves the one-shot auto-open alone on
// failure, shows the real error and lets the user retry explicitly.
func TestViewRetriesTheHostAfterAnError(t *testing.T) {
	webviewTestLocale(t)
	opened := 0
	m := Model{
		HarnessOrigin: "http://127.0.0.1:3080",
		ThemeID:       "deepseek",
		HostError:     "WebView2 runtime unavailable",
		OnOpenHost:    func() { opened++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	if opened != 0 {
		t.Fatalf("auto-opened despite an error: %d", opened)
	}
	if !tt.HasText("WebView2 runtime unavailable") {
		t.Fatalf("missing real error: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyWebViewOpenHost)); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Fatalf("retry opens = %d, want 1", opened)
	}
}

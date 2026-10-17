package browserpreview

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestViewBrowserPreviewLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := "example.com"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:    "deepseek",
			URLDraft:   &draft,
			HostActive: true,
			HostURL:    "https://example.com/",
			OnNavigate: func(string) {},
			OnOpenExternal: func(string) {},
			OnCloseHost: func() {},
		})
	}, 720, 480)

	want := []string{
		i18n.T(i18n.KeyBrowserPreviewTitle),
		i18n.T(i18n.KeyBrowserPreviewGo),
		i18n.T(i18n.KeyBrowserPreviewOpenExternal),
		i18n.T(i18n.KeyBrowserPreviewCloseHost),
		"https://example.com/",
	}
	for _, s := range want {
		if !tt.HasText(s) {
			t.Fatalf("missing %q; texts=%q", s, tt.Texts())
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	if got := NormalizeURL("example.com/path"); got != "https://example.com/path" {
		t.Fatalf("got %q", got)
	}
	if NormalizeURL("ftp://x") != "" {
		t.Fatal("expected reject ftp")
	}
	if NormalizeURL("  ") != "" {
		t.Fatal("expected empty")
	}
}

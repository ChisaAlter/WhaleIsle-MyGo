package im

import (
	"testing"

	"github.com/egoist/mygo/ui"

	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/ui/i18n"
)

func TestViewIMLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	store, err := imstore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	draft := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Store:            store,
			ThemeID:          "deepseek",
			HarnessReady:     true,
			MuxConnected:     true,
			HarnessIMOverlay: true,
			Draft:            &draft,
			OnOpenPluginWeb:  func() {},
		})
	}, 800, 600)

	want := []string{
		i18n.T(i18n.KeyIMTitle),
		i18n.T(i18n.KeyIMChannels),
		i18n.T(i18n.KeyIMHistory),
		i18n.T(i18n.KeyIMOpenWeb),
		i18n.T(i18n.KeyIMLocalMode),
		i18n.T(i18n.KeyIMBridge),
	}
	for _, s := range want {
		if !tt.HasText(s) {
			t.Fatalf("missing %q", s)
		}
	}
}

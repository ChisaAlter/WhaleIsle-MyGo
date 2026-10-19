package market

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	marketstore "whaleisle/internal/store/market"
)

func TestPluginArtworkDynamicSVGWithoutOwnerRetainsRealConsumerFallback(t *testing.T) {
	body := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="30" height="20"><rect width="30" height="20"><animate attributeName="opacity" values="0;1" dur="1s"/></rect></svg>`)
	cache := readmeImageCache(t)
	cache.SetHTTPClient(&http.Client{Transport: readmeImageTransport(func(*http.Request) (*http.Response, error) {
		t.Error("cached fixture attempted HTTP")
		return nil, errors.New("fixture forbids network")
	})})
	remote := "https://example.invalid/animated-meta.svg"
	if err := os.WriteFile(cache.FilePath(remote), body, 0600); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{remote, "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(body)} {
		accepted := true
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				accepted = PluginImage(c, source, 30, 20)
				if !accepted {
					ui.Text(c, "Default plugin artwork")
				}
			})
		}, 160, 100)
		if accepted || !tt.HasText("Default plugin artwork") {
			t.Fatalf("dimensions-only SVG suppressed the fallback for %s", source)
		}
	}
}

func TestDesktopInstalledProjectionRetainsPartialDependenciesOnly(t *testing.T) {
	disabled := false
	rows := []marketstore.Entry{
		{Label: "saved-awaiting-restart", IdentityKnown: true, Installed: true, Enabled: &disabled, Error: "restart required"},
		{Label: "missing-files", IdentityKnown: true, Installed: true, ReadOnlyReason: "files missing"},
		{Label: "runtime-optional", IdentityKnown: true, Installed: false},
		{Label: "identity-not-read", Installed: true},
	}
	got := desktopInstalledRows(rows)
	if len(got) != 2 || got[0].Label != "saved-awaiting-restart" || got[1].Label != "missing-files" || got[0].Enabled == nil || *got[0].Enabled {
		t.Fatalf("partial dependency state was confused with runtime availability: %+v", got)
	}
}

package boot

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestPluginProgressLine(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	got := pluginProgressLine(harness.Status{
		State: harness.Starting,
		PluginLoad: harness.PluginProgress{Phase: "runtime", Ready: 2, Total: 5, Detail: "web-app"},
	})
	want := fmt.Sprintf(i18n.T(KeyPlugins), 2, 5) + " — " + fmt.Sprintf(i18n.T(KeyPluginsDetail), "web-app")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStatusLine(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	if got := statusLine(harness.Status{State: harness.Starting}); got != i18n.T(KeyStarting) {
		t.Fatalf("starting = %q", got)
	}
	if got := statusLine(harness.Status{State: harness.Stopped}); got != i18n.T(KeyStopped) {
		t.Fatalf("stopped = %q", got)
	}
}

// TestViewBootLabels renders the boot page headlessly and checks status + actions.
func TestViewBootLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	progress := fmt.Sprintf(i18n.T(KeyPlugins), 1, 3)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Status: harness.Status{
				State:      harness.Starting,
				PluginLoad: harness.PluginProgress{Phase: "runtime", Ready: 1, Total: 3},
			},
			ThemeID:    "system",
			SystemDark: false,
			OnRetry:    func() {},
			OnExport:   func() {},
		})
	}, 800, 600)

	want := []string{
		i18n.T(KeyTitle),
		i18n.T(KeyStarting),
		progress,
		i18n.T(KeyRetry),
		i18n.T(KeyExportLog),
		i18n.T(KeyWaiting),
	}
	for _, label := range want {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts = %q", label, tt.Texts())
		}
	}
}

package boot

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/product"
	"whaleisle/internal/ui/i18n"
)

func TestPluginProgressLine(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	got := pluginProgressLine(harness.Status{
		State:      harness.Starting,
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

// TestViewBootLabels renders the boot page headlessly and checks the brand block,
// the status line, the actions and the foot ticker.
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

	// The reference brand block (boot.html:22) replaces the 24 DIP title with the
	// serif wordmark, the island's mark and the harness line; the foot ticker
	// counts the log lines it holds.
	want := []string{
		product.Name,
		brandMark,
		"BASED ON DEEPSEEK HARNESS",
		i18n.T(KeyStarting),
		progress,
		i18n.T(KeyWaiting),
		"L 00",
	}
	for _, label := range want {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts = %q", label, tt.Texts())
		}
	}
	if tt.HasText(i18n.T(KeyRetry)) || tt.HasText(i18n.T(KeyExportLog)) {
		t.Fatal("normal starting exposed recovery actions")
	}
}

// TestViewBootTickerCountsLogs pins the ticker's L 00 counter against the log
// tail it shows (boot.js updateLogCount).
func TestViewBootTickerCountsLogs(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Status:  harness.Status{State: harness.Starting},
			Logs:    []string{"first evidence", "latest evidence"},
			ThemeID: "system",
		})
	}, 800, 600)
	if !tt.HasText("L 02") {
		t.Fatalf("missing log count; texts = %q", tt.Texts())
	}
	if !tt.HasText("latest evidence") {
		t.Fatalf("ticker lost the newest line; texts = %q", tt.Texts())
	}
}

func TestCancelledRecoveryStatusAndAction(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, active := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Status: harness.Status{State: harness.Starting, RecoveryActive: active, RecoveryScheduled: active}, ThemeID: "system", OnCancelRecovery: func() {}})
		}, 800, 600)
		if tt.HasText("Cancel automatic recovery") != active {
			t.Fatal("cancel visibility does not match recovery")
		}
	}
	if got := statusLine(harness.Status{State: harness.Failed, RecoveryCancelled: true}); got != "Automatic recovery cancelled. You can retry manually." {
		t.Fatalf("cancelled status = %q", got)
	}
}

func TestRecoveryControlHiddenOutsideRecovery(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, st := range []harness.Status{
		{State: harness.Ready},
		{State: harness.Starting, RecoveryActive: true},
		{State: harness.Starting, RecoveryActive: true, RecoveryScheduled: true},
		{State: harness.Failed, RecoveryCancelled: true},
	} {
		tt := ui.NewTester(func(c *ui.Context) {
			RecoveryControl(c, Model{Status: st, ThemeID: "system", OnCancelRecovery: func() {}, OnRetry: func() {}})
		}, 800, 150)
		if tt.HasText("Cancel automatic recovery") != st.RecoveryScheduled {
			t.Fatal("mounted cancel action visibility")
		}
		if tt.HasText(i18n.T(KeyRetry)) != (st.State == harness.Failed || st.RecoveryActive || st.RecoveryCancelled) {
			t.Fatal("mounted manual retry visibility")
		}
		if st.State == harness.Ready && len(tt.Texts()) != 0 {
			t.Fatal("normal dialogue gained a boot control")
		}
	}
}

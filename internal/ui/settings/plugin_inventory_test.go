package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginslots"
)

func TestSP43ClientModulesStatusAndRetry(t *testing.T) {
	retries, refreshes := 0, 0
	access := &PluginInventoryAccess{OnRefresh: func() { refreshes++ }}
	access.OnRetryClientModules = func() { retries++; access.ClientModules.Retrying = true }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			pluginInventoryPane(c, access, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80))
			// The client modules moved under the package-management disclosure.
			clientModulesPane(c, access, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80))
		})
	}, 1000, 900)
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), pluginsText("客户端页面未连接。", "Client page disconnected.")) {
		t.Fatal("no-page state was not explicit")
	}
	retryLabel := pluginsText("重试客户端模块", "Retry client modules")
	if _, ok := tt.Find(retryLabel); ok {
		t.Fatal("disconnected page offered retry")
	}
	access.ClientPage = "http://127.0.0.1:3080/?desktopPlugin=fixture"
	access.ClientModules = pluginslots.ClientModulesSnapshot{Page: 7, Connected: true, Failures: []pluginslots.ClientModuleFailure{{ID: "broken-module", Message: "apply failed"}}}
	tt.Frame()
	text := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{access.ClientPage, "broken-module: apply failed"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing page state %q: %s", want, text)
		}
	}
	if err := tt.Click(retryLabel); err != nil {
		t.Fatal(err)
	}
	if retries != 1 || refreshes != 0 {
		t.Fatal("client retry used Host inventory refresh")
	}
	_ = tt.Click(retryLabel)
	if retries != 1 {
		t.Fatal("pending retry was clickable")
	}
	access.ClientModules = pluginslots.ClientModulesSnapshot{}
	tt.Frame()
	text = strings.Join(tt.Texts(), "\n")
	if strings.Contains(text, "apply failed") || strings.Contains(text, access.ClientPage) {
		t.Fatal("closed page retained errors or ownership")
	}
}

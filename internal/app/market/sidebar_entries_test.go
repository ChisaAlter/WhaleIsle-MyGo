package marketctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
)

type stubHostDeps struct {
	invalidated atomic.Int32
}

func (s *stubHostDeps) Env() HostEnvSnapshot              { return HostEnvSnapshot{} }
func (s *stubHostDeps) LifecycleContext() context.Context { return context.Background() }
func (s *stubHostDeps) GoPluginInstallRegistry() (*config.PluginInstallRegistry, error) {
	return nil, nil
}
func (s *stubHostDeps) SaveGoPluginInstallRegistry(*config.PluginInstallRegistry) error { return nil }
func (s *stubHostDeps) SyncHarnessAndRestart()                                          {}
func (s *stubHostDeps) UpdateWindow(fn func()) {
	if fn != nil {
		fn()
	}
}
func (s *stubHostDeps) InvalidateWindow() { s.invalidated.Add(1) }
func (s *stubHostDeps) CloseWebViewHost() {}

func TestCoordinatorSidebarPluginsZeroWebViewOnAttach(t *testing.T) {
	profileDir := t.TempDir()
	qaSrc := filepath.Join("..", "..", "..", "cmd", "whaleisle", "testdata", "qa-panel")
	qaDst := filepath.Join(profileDir, "node_modules", "qa-panel")
	if err := os.MkdirAll(filepath.Join(qaDst, "client"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"package.json", filepath.Join("client", "client.js")} {
		data, err := os.ReadFile(filepath.Join(qaSrc, rel))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(qaDst, rel), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env map[string]any
		_ = json.NewDecoder(r.Body).Decode(&env)
		var value any = map[string]any{}
		switch r.URL.Path {
		case "/api/pluginInventory/list":
			value = map[string]any{
				"managementAvailable": true,
				"entries": []map[string]any{
					{"entryId": "qa-panel", "moduleName": "qa-panel", "enabled": true, "fiberPhase": "active"},
				},
			}
		case "/api/pluginManager/listBundles":
			value = []map[string]any{
				{"name": "qa-panel", "installed": true, "enabled": true, "removable": true, "official": false},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": env["rpcId"], "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()

	deps := &stubHostDeps{}
	coord := NewCoordinator(deps)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	coord.AttachHost(client, profileDir, nil)
	defer coord.DetachHost()

	waitMarketFixture(t, func() bool {
		got := coord.SidebarPlugins("zh-CN")
		return len(got) == 1 && got[0].ID == "qa-panel" && got[0].Label == "QA Panel"
	})
}

package pluginmanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestActionsUpdateBlockedWhileInstallOrBusyActive(t *testing.T) {
	var removeCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path == "/api/pluginManager/removeBundle" || r.URL.Path == "/api/pluginManager/remove" {
			removeCalls.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  body["rpcId"],
			"result": map[string]any{"ok": true, "value": map[string]any{"application": "applied"}},
		})
	}))
	defer srv.Close()

	c := New()
	c.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	owner := c.Snapshot().Value.Owner

	pkg := Package{
		Bundle: Bundle{
			Name:      "demo-plugin",
			Installed: true,
			Removable: true,
			Enabled:   true,
			Version:   "1.0.0",
			InstallTarget: &InstallTarget{
				Spec:    "demo-plugin@2.0.0",
				Version: "2.0.0",
			},
		},
	}

	c.mu.Lock()
	c.state.Packages = []Package{pkg}
	c.state.Install = InstallState{Open: true, RequestID: "req-active", Phase: "installing"}
	c.unlockPublish()

	c.Actions(owner).Update("demo-plugin")
	time.Sleep(30 * time.Millisecond)
	if removeCalls.Load() != 0 {
		t.Fatalf("Actions.Update called Remove while active Install in progress (%d calls)", removeCalls.Load())
	}

	c.mu.Lock()
	c.state.Install = InstallState{Phase: "idle"}
	c.state.Busy = true
	c.unlockPublish()

	c.Actions(owner).Update("demo-plugin")
	time.Sleep(30 * time.Millisecond)
	if removeCalls.Load() != 0 {
		t.Fatalf("Actions.Update called Remove while controller Busy (%d calls)", removeCalls.Load())
	}
}

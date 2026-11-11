package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestACPActionSnapshotsDraftAndDiscardsOldHostResponse(t *testing.T) {
	entered, release := make(chan map[string]any, 1), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		entered <- req
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"agents": []any{map[string]any{"id": "old-host"}}}}})
	}))
	defer server.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}
	s := &mainState{dshClient: client, acpPanel: acpPanelState{owner: client}}
	env := map[string]string{"KEY": "owned-original"}
	s.runACP("agent/save", map[string]any{"agent": map[string]any{"env": env}})
	env["KEY"] = "edited-after-dispatch"
	select {
	case req := <-entered:
		got := req["payload"].(map[string]any)["agent"].(map[string]any)["env"].(map[string]any)["KEY"]
		if got != "owned-original" {
			close(release)
			t.Fatal("in-flight request read mutable draft", got)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("no ACP request")
	}
	s.dshMu.Lock()
	s.dshClient = nil
	s.dshMu.Unlock()
	s.acpSettingsAccess() // invalidates/cancels old ownership before response arrives
	close(release)
	// Wait for the request's cancellation; it must not repopulate the new owner.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		p := s.acpPanel
		s.mu.Unlock()
		if p.owner != nil || p.busy || len(p.catalog.Agents) != 0 || p.status != "" {
			t.Fatal("old Host polluted ACP panel", p)
		}
		time.Sleep(time.Millisecond)
	}
}

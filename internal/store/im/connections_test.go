package im

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

func TestConnectionManagementFixture(t *testing.T) {
	var mu sync.Mutex
	workspace, preset := "/fixture/old", ""
	calls := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string         `json:"rpcId"`
			Payload map[string]any `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, r.URL.Path)
		if r.URL.Path != "/telegram/connection.status" && req.Payload["botId"] != "fixture-bot" {
			t.Error("missing exact bot id")
		}
		switch r.URL.Path {
		case "/telegram/bot.reconnect":
			if req.Payload["sendTest"] != false {
				t.Error("reconnect must never send test message")
			}
		case "/telegram/bot.workspace.set":
			workspace = req.Payload["workspace"].(string)
		case "/telegram/bot.preset.set":
			preset, _ = req.Payload["agentPreset"].(string)
		case "/telegram/connection.status":
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
		}
		value := map[string]any{"snapshot": map[string]any{
			"bots":               []any{map[string]any{"botId": "fixture-bot", "state": "connected", "connected": true, "workspace": workspace, "agentPreset": preset, "bot": map[string]any{"name": "Fixture"}}},
			"agentPresetCatalog": map[string]any{"items": []any{"general", map[string]any{"id": "coding", "name": "Coding"}}},
		}}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	s := New()
	s.AttachPlugin(&pluginrpc.Client{BaseURL: srv.URL})
	s.rows = parseCordisBots(json.RawMessage(`{"bots":[{"botId":"fixture-bot","workspace":"/fixture/old"}],"agentPresetCatalog":{"items":["coding"]}}`), "telegram")
	s.selectedID = "telegram:fixture-bot"
	await := func() {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for s.Snapshot().ConnectionBusy && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if s.Snapshot().ConnectionBusy || s.Snapshot().ConnectionNote != storelocale.T(storelocale.KeyStoreImBotStatusUpdated) {
			t.Fatalf("operation failed: %+v", s.Snapshot())
		}
	}
	if err := s.SaveWorkspace(context.Background(), s.selectedID, "relative"); err == nil {
		t.Fatal("relative path accepted")
	}
	if err := s.SaveAgentPreset(context.Background(), s.selectedID, "not-installed"); err == nil {
		t.Fatal("unknown preset accepted")
	}
	if err := s.ReconnectBot(context.Background(), s.selectedID); err != nil {
		t.Fatal(err)
	}
	await()
	s.SetWorkspaceDraft(s.selectedID, "/fixture/new")
	if err := s.SaveWorkspace(context.Background(), s.selectedID, "/fixture/new"); err != nil {
		t.Fatal(err)
	}
	await()
	if s.Snapshot().WorkspaceDraft != "/fixture/new" {
		t.Fatal("workspace did not roundtrip")
	}
	if err := s.SaveAgentPreset(context.Background(), s.selectedID, "coding"); err != nil {
		t.Fatal(err)
	}
	await()
	if s.Snapshot().Rows[0].AgentPreset != "coding" {
		t.Fatal("preset did not roundtrip")
	}
	if err := s.SaveAgentPreset(context.Background(), s.selectedID, ""); err != nil {
		t.Fatal(err)
	}
	await()
	if s.Snapshot().Rows[0].AgentPreset != "" {
		t.Fatal("preset null did not roundtrip")
	}
	snap := s.Snapshot()
	snap.Rows[0].Presets[0].Label = "modified"
	if s.Snapshot().Rows[0].Presets[0].Label == "modified" {
		t.Fatal("snapshot aliases preset catalog")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 8 {
		t.Fatalf("unexpected calls %v", calls)
	}
	for _, call := range calls {
		if strings.Contains(call, "send") || strings.Contains(call, "credentials") {
			t.Fatal("unsafe call")
		}
	}
}

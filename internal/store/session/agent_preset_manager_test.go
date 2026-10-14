package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/dsh"
)

func TestAgentPresetManagerPersistDefaultFixture(t *testing.T) {
	var reject atomic.Bool
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   any `json:"rpcId"`
			Payload struct {
				Args map[string]json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/agentPresets/list"):
			rpcOK(w, body.RPCID, map[string]any{"presets": []map[string]any{{"id": "standard", "isDefault": true}, {"id": "coder", "name": "Coder"}, {"id": "broken", "broken": "missing config"}}})
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body.RPCID, map[string]any{"writable": true, "namespaces": []map[string]any{{"ns": "agent-preset-registry", "revision": 7, "value": map[string]any{"selectedDefault": "standard"}}}})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			writes.Add(1)
			if string(body.Payload.Args["ns"]) != `"agent-preset-registry"` || string(body.Payload.Args["expectedRevision"]) != "7" {
				t.Errorf("bad args: %+v", body.Payload.Args)
			}
			if reject.Load() {
				http.Error(w, "fixture refusal", 409)
				return
			}
			var patch struct {
				SelectedDefault string `json:"selectedDefault"`
			}
			_ = json.Unmarshal(body.Payload.Args["patch"], &patch)
			rpcOK(w, body.RPCID, map[string]any{"ns": "agent-preset-registry", "revision": 8, "value": map[string]any{"selectedDefault": patch.SelectedDefault}})
		default:
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	manager := NewAgentPresetManager(s)
	manager.Refresh(context.Background())
	if got := manager.Snapshot(); got.DefaultID != "standard" || len(got.Rows) != 3 || !got.Writable {
		t.Fatalf("load: %+v", got)
	}
	manager.SaveDefault(context.Background(), "broken")
	if writes.Load() != 0 || manager.Snapshot().Err == "" {
		t.Fatal("broken preset should be refused before write")
	}
	reject.Store(true)
	manager.SaveDefault(context.Background(), "coder")
	if got := manager.Snapshot(); got.DefaultID != "standard" || got.Err == "" {
		t.Fatalf("failed write changed default: %+v", got)
	}
	reject.Store(false)
	manager.SaveDefault(context.Background(), "coder")
	if got := manager.Snapshot(); got.DefaultID != "coder" || got.Err != "" || got.Revision != 8 {
		t.Fatalf("save: %+v", got)
	}
	if s.Snapshot().SelectedAgentPreset != "" {
		t.Fatal("default save must not change active session preset")
	}
}

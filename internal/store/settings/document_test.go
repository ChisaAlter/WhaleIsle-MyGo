package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestIsDocumentUpdatedEmit(t *testing.T) {
	if !IsDocumentUpdatedEmit("settings/document-updated", "") {
		t.Fatal("document-updated")
	}
	if !IsDocumentUpdatedEmit("settings.updated", `{"ns":"x"}`) {
		t.Fatal("settings updated")
	}
	if IsDocumentUpdatedEmit("session/follow", "") {
		t.Fatal("unrelated")
	}
}

func TestOpenSettingsDocument(t *testing.T) {
	var mu sync.Mutex
	opened := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/openSettingsDocument"):
			mu.Lock()
			opened = true
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": env.RPCID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": env.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"writable":   true,
						"namespaces": []any{},
					},
				},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.OpenSettingsDocument(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ready := opened && st.Snapshot().LastAction == "settings document opened"
		mu.Unlock()
		if ready {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	seen, action := opened, st.Snapshot().LastAction
	mu.Unlock()
	t.Fatalf("opened=%v action=%q", seen, action)
}

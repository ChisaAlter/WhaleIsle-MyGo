package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestRefreshListPopulatesItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/session/list" {
			t.Fatalf("path %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"items": []map[string]any{{
						"sessionId":     "sess-1",
						"updatedAt":     1.0,
						"running":       false,
						"blank":         true,
						"agentAvailable": true,
					}},
				},
			},
		})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.ListLoading || len(snap.Items) == 0 {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})

	st.RefreshList(context.Background())
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-waitCtx.Done():
		t.Fatal("timeout waiting for list refresh")
	}

 snap := st.Snapshot()
	if len(snap.Items) != 1 || snap.Items[0].SessionID != "sess-1" {
		t.Fatalf("items = %+v", snap.Items)
	}
	if snap.SelectedID != "sess-1" {
		t.Fatalf("selected = %q", snap.SelectedID)
	}
}

func TestTitleFromProjections(t *testing.T) {
	raw := json.RawMessage(`{"title":"Plan migration"}`)
	if got := titleFromProjections(raw); got != "Plan migration" {
		t.Fatalf("got %q", got)
	}
}

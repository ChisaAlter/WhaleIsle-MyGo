package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

func TestHandleModelCatalogEventDispatchesAllHostEvents(t *testing.T) {
	var initCalls atomic.Int32
	var catalogCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/initializeDefaultModel"):
			initCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": body["rpcId"],
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		case strings.HasSuffix(r.URL.Path, "/session/modelCatalog"):
			catalogCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": body["rpcId"],
				"result": map[string]any{"ok": true, "value": map[string]any{
					"default": map[string]any{"provider": "deepseek-official", "model": "deepseek-flash"},
					"groups":  []any{},
				}},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": body["rpcId"],
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		}
	}))
	defer srv.Close()

	store := session.New()
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: srv.URL})
	defer client.Close()
	store.Attach(client, nil)
	defer store.Detach()

	waitCount := func(name string, counter *atomic.Int32, want int32) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if counter.Load() >= want {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("%s count = %d, want >= %d", name, counter.Load(), want)
	}

	// Non-existent legacy event name must not trigger catalog RPCs.
	handleModelCatalogEvent(context.Background(), store, "llm/providers-updated")
	time.Sleep(30 * time.Millisecond)
	if initCalls.Load() != 0 || catalogCalls.Load() != 0 {
		t.Fatalf("unexpected RPCs for removed llm/providers-updated: init=%d catalog=%d", initCalls.Load(), catalogCalls.Load())
	}

	// credentials/record-updated seeds default model and chains modelCatalog.
	handleModelCatalogEvent(context.Background(), store, "credentials/record-updated")
	waitCount("initializeDefaultModel", &initCalls, 1)
	waitCount("modelCatalog after record-updated", &catalogCalls, 1)

	// The remaining 4 real Host 0.2.2 events refresh modelCatalog directly.
	var wantCatalog int32 = 1
	for _, ev := range []string{
		"llm/adapters-updated",
		"llm/models-updated",
		"settings/document-updated",
		"credentials/reference-updated",
	} {
		wantCatalog++
		handleModelCatalogEvent(context.Background(), store, ev)
		waitCount("modelCatalog for "+ev, &catalogCalls, wantCatalog)
	}
	if got := initCalls.Load(); got != 1 {
		t.Fatalf("initializeDefaultModel count = %d, want 1 (only credentials/record-updated)", got)
	}

	// account/watch credential-stored also calls InitializeDefaultModel; sign-out calls RefreshModelCatalog.
	handleAccountCredentialsChanged(context.Background(), store, "credential-stored")
	waitCount("initializeDefaultModel after credential-stored", &initCalls, 2)
	waitCount("modelCatalog after credential-stored", &catalogCalls, wantCatalog+1)

	wantCatalog++
	handleAccountCredentialsChanged(context.Background(), store, "signed-out")
	waitCount("modelCatalog after signed-out", &catalogCalls, wantCatalog+1)
	if got := initCalls.Load(); got != 2 {
		t.Fatalf("signed-out unexpectedly called initializeDefaultModel: count=%d", got)
	}
}

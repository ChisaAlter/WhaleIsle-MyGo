package acp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestPatchPreservesConfigurationAndUnknownFields(t *testing.T) {
	var a Agent
	if err := json.Unmarshal([]byte(`{"id":"local","models":[{"id":"discovered"}],"configuredModels":[{"id":"saved","reasoning":{"efforts":["high"]}}],"favoriteModels":["saved"],"hiddenModels":["hidden"],"driverKind":"acp","futureField":{"value":2}}`), &a); err != nil {
		t.Fatal(err)
	}
	patch := a.Patch(map[string]any{"name": "New name"})
	for _, key := range []string{"driverKind", "futureField", "favoriteModels", "hiddenModels"} {
		if string(patch[key]) != string(a.Raw[key]) {
			t.Fatal(key)
		}
	}
	if string(patch["models"]) != string(a.ConfiguredModels) {
		t.Fatal("effective discovered models replaced saved config")
	}
	patch["futureField"][0] = 'x'
	if a.Raw["futureField"][0] == 'x' {
		t.Fatal("patch aliases catalog data")
	}
}
func TestActionConsumesWrappedCatalogAndRejectsStaleOrFailedAction(t *testing.T) {
	value := map[string]any{"catalog": map[string]any{"agents": []any{map[string]any{"id": "custom"}}}, "createdId": "custom"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "owned=qa" {
			t.Error("missing Host authentication")
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": value}})
	}))
	defer server.Close()
	c := Client{RPC: &pluginrpc.Client{BaseURL: server.URL, Cookie: "owned=qa"}}
	catalog, id, err := c.Action(context.Background(), "agent/create-account", nil)
	if err != nil || id != "custom" || len(catalog.Agents) != 1 {
		t.Fatal(catalog, id, err)
	}
	value = map[string]any{"sent": false, "catalog": catalog}
	if _, _, err = c.Action(context.Background(), "agent/login/respond", nil); err == nil {
		t.Fatal("stale interaction accepted")
	}
	value = map[string]any{"ok": false, "error": "handshake failed"}
	if _, _, err = c.Action(context.Background(), "agent/test", nil); err == nil {
		t.Fatal("failed probe accepted")
	}
}

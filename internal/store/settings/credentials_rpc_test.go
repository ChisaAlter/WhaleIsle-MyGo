package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestSyncAPIKeyToHarnessSetAndUnset(t *testing.T) {
	var lastMethod, lastRef, lastValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		lastMethod = r.URL.Path
		if strings.HasSuffix(lastMethod, "/settings/describe") {
			rpcOK(w, body["rpcId"], map[string]any{"namespaces": []any{}})
			return
		}
		payload, _ := body["payload"].(map[string]any)
		args, _ := payload["args"].(map[string]any)
		if args != nil {
			lastRef, _ = args["ref"].(string)
			lastValue, _ = args["value"].(string)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok":    true,
				"value": map[string]any{},
			},
		})
	}))
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	if err := SyncAPIKeyToHarness(context.Background(), client, "sk-test"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(lastMethod, "/credentials/set") || lastRef != CredRefAPIKey || lastValue != "sk-test" {
		t.Fatalf("set method=%s ref=%s value=%s", lastMethod, lastRef, lastValue)
	}
	if err := SyncAPIKeyToHarness(context.Background(), client, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(lastMethod, "/credentials/unset") || lastRef != CredRefAPIKey {
		t.Fatalf("unset method=%s ref=%s", lastMethod, lastRef)
	}
}

func TestSyncAPIKeyUsesAcceptedHostReference(t *testing.T) {
	ref := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/settings/describe") {
			rpcOK(w, body["rpcId"], map[string]any{"namespaces": []any{map[string]any{"ns": "llm-deepseek-api-key", "value": map[string]any{"apiKeyEnv": "OWNED_DEEPSEEK_KEY"}}}})
			return
		}
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		ref, _ = args["ref"].(string)
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()
	if err := SyncAPIKeyToHarness(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, "synthetic-key"); err != nil {
		t.Fatal(err)
	}
	if ref != "OWNED_DEEPSEEK_KEY" || CredRefAPIKey != "DEEPSEEK_API_KEY" {
		t.Fatalf("wrong adapter reference: %s", ref)
	}
}

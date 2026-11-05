package pluginrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallRejectsWrongResponseIdentityBeforeResult(t *testing.T) {
	for _, mode := range []string{"type", "rpcId"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var req clientRequest
				json.NewDecoder(r.Body).Decode(&req)
				kind, id := "server-response", req.RPCID
				if mode == "type" {
					kind = "client-request"
				} else {
					id = "another-request"
				}
				json.NewEncoder(w).Encode(map[string]any{"type": kind, "rpcId": id, "result": map[string]any{"ok": true, "value": map[string]any{"mustNotBeConsumed": true}}})
			}))
			defer srv.Close()
			var out map[string]any
			err := (&Client{BaseURL: srv.URL}).Call(context.Background(), "dsh-whale", "catalog", map[string]any{}, &out)
			if err == nil || !strings.Contains(err.Error(), "identity") || out != nil || calls != 1 {
				t.Fatal("mismatched envelope consumed", err, out, calls)
			}
		})
	}
}

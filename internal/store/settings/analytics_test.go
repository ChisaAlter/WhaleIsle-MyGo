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

func TestProductAnalyticsEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/productAnalytics/enabled") {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok":    true,
				"value": true,
			},
		})
	}))
	defer srv.Close()
	on, err := ProductAnalyticsEnabled(context.Background(), &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}})
	if err != nil || !on {
		t.Fatalf("on=%v err=%v", on, err)
	}
}

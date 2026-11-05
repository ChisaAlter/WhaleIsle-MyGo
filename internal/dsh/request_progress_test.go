package dsh

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallWithProgressCountsActualAuthenticatedBody(t *testing.T) {
	received := int64(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		received = int64(len(raw))
		if r.Header.Get("Cookie") != "scope=test" || r.ContentLength != received {
			t.Error("transport contract changed")
		}
		var b map[string]any
		_ = json.Unmarshal(raw, &b)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": b["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"accepted": true}}})
	}))
	defer server.Close()
	c := Client{Endpoint: Endpoint{Origin: server.URL, Cookie: "scope=test"}}
	var sent, total, last int64
	var out struct{ Accepted bool }
	err := c.CallWithProgress(context.Background(), "fileUploads/upload", map[string]any{"data": strings.Repeat("x", 128<<10)}, &out, func(n, size int64) {
		if n < last || n > size {
			t.Error("non-monotone wire progress")
		}
		sent, total, last = n, size, n
	})
	if err != nil || !out.Accepted || sent != received || total != received || received <= 128<<10 {
		t.Fatalf("progress sent=%d total=%d received=%d err=%v", sent, total, received, err)
	}
}

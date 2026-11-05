package settings

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
)

// One keyless fixture exercises the catalog/read/write/error/disable contract.
func TestVisionSettingsContract(t *testing.T) {
	var writes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{"writable": true, "namespaces": []any{map[string]any{"ns": VisionNamespace, "revision": 7, "schema": map[string]any{"type": "object"}, "value": map[string]any{"provider": "stale", "model": "old", "timeoutMs": 123}}}})
		case strings.HasSuffix(r.URL.Path, "/session/modelCatalog"):
			rpcOK(w, body["rpcId"], map[string]any{"groups": []any{map[string]any{"id": "p", "name": "P", "models": []any{map[string]any{"id": "image", "name": "Image", "inputModalities": []string{"image"}}, map[string]any{"id": "text", "name": "Text", "inputModalities": []string{"text"}}}}}})
		case strings.HasSuffix(r.URL.Path, "/settings/mutate"):
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			if args["ns"] != VisionNamespace || args["expectedRevision"] != float64(7) {
				t.Errorf("wrong live/global request: %v", args)
			}
			ops := args["ops"].([]any)
			if len(ops) != 2 {
				t.Errorf("must change only route: %v", ops)
			}
			n := writes.Add(1)
			if n == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": false, "error": map[string]any{"message": "write denied"}}})
				return
			}
			for _, op := range ops {
				m := op.(map[string]any)
				if m["op"] != "unset" || m["value"] != nil {
					t.Errorf("disable must remove the user route overrides: %v", op)
				}
			}
			rpcOK(w, body["rpcId"], map[string]any{"ns": VisionNamespace, "revision": 8, "value": map[string]any{"timeoutMs": 123}})
		default:
			t.Errorf("unexpected RPC: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	await := func(done func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !done() && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !done() {
			t.Fatal("fixture timeout")
		}
	}
	st.RefreshVision(context.Background())
	await(func() bool { return !st.VisionSnapshot().Loading })
	v := st.VisionSnapshot()
	if !v.Present || v.Provider != "stale" || len(v.Routes) != 1 || v.Routes[0].Model != "image" {
		t.Fatalf("catalog/current selection: %+v", v)
	}
	st.selectedNS = "unrelated"
	st.SaveVision(context.Background(), "p", "image")
	await(func() bool { return !st.VisionSnapshot().Saving })
	if st.VisionSnapshot().Err == "" || st.VisionSnapshot().Provider != "stale" {
		t.Fatal("failed save must retain current selection and show error")
	}
	st.SaveVision(context.Background(), "", "")
	await(func() bool { return !st.VisionSnapshot().Saving })
	v = st.VisionSnapshot()
	if v.Err != "" || v.Provider != "" || v.Model != "" || st.selectedNS != "unrelated" {
		t.Fatalf("disable/global scope: %+v", v)
	}
	var values map[string]any
	_ = json.Unmarshal(st.namespaces[0].Value, &values)
	if values["timeoutMs"] != float64(123) || writes.Load() != 2 {
		t.Fatal("unrelated settings changed")
	}
}

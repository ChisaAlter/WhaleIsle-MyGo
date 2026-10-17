package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestKillJobCallsRPC(t *testing.T) {
	var gotSession, gotJob string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/job/kill") {
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				gotSession, _ = req["sessionId"].(string)
				gotJob, _ = req["jobId"].(string)
			}
			writeRPC(w, body["rpcId"], map[string]any{"outcome": "killed"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/job/list") {
			writeRPC(w, body["rpcId"], map[string]any{"type": "snapshot", "jobs": []any{}})
			return
		}
		writeRPC(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.KillJob(context.Background(), "s1", "j9")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if gotSession == "s1" && gotJob == "j9" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session/job = %q/%q", gotSession, gotJob)
}

func writeRPC(w http.ResponseWriter, rpcID any, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":  "server-response",
		"rpcId": rpcID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	})
}

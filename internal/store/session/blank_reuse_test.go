package session

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

func TestCreateSessionBlankReuseRPC(t *testing.T) {
	var hitReuse atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/blankReuse") {
			hitReuse.Store(true)
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req == nil || req["sessionId"] != "blank-1" {
				t.Errorf("unexpected blankReuse args: %#v", args)
			}
			rpcOK(w, body["rpcId"], map[string]any{"reusable": true})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	st := New()
	st.mu.Lock()
	st.cwd = "/ws"
	st.items = []Item{{SessionID: "blank-1", Blank: true, Cwd: "/ws"}}
	st.mu.Unlock()
	reused := make(chan struct{}, 1)
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if snap := st.Snapshot(); snap.SelectedID == "blank-1" && snap.LastAction == "blank session reused" {
			select {
			case reused <- struct{}{}:
			default:
			}
		}
	})
	st.CreateSession(context.Background())
	select {
	case <-reused:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for async blank reuse: selected=%q action=%q", st.Snapshot().SelectedID, st.Snapshot().LastAction)
	}
	if !hitReuse.Load() {
		t.Fatal("expected session/blankReuse")
	}
	if st.Snapshot().SelectedID != "blank-1" {
		t.Fatalf("selected = %q", st.Snapshot().SelectedID)
	}
	if st.Snapshot().LastAction != "blank session reused" {
		t.Fatalf("action = %q", st.Snapshot().LastAction)
	}
}

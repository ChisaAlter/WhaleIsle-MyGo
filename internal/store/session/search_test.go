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

func TestSearchSessionsHits(t *testing.T) {
	var searchHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/session/search"):
			searchHits.Add(1)
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req["query"] != "hello" {
				t.Fatalf("query = %v", req["query"])
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"items": []map[string]any{{
					"sessionId": "sess-remote",
					"snippet":   "… hello …",
				}},
				"hasMore": false,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)
	st.mu.Lock()
	st.items = []Item{
		{SessionID: "sess-remote", Title: "Other title"},
		{SessionID: "sess-local", Title: "hello local"},
	}
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.SearchLoading || snap.SearchRPCQuery != "hello" {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})

	st.UpdateSessionSearch(context.Background(), "hello")
	wait(t, done)

	if searchHits.Load() != 1 {
		t.Fatalf("search RPC hits = %d", searchHits.Load())
	}
	snap := st.Snapshot()
	if snap.SearchUseLocal {
		t.Fatal("expected RPC hits, not local fallback")
	}
	if !snap.SearchHitIDs["sess-remote"] {
		t.Fatalf("hits = %+v", snap.SearchHitIDs)
	}
}

func TestSearchSessionsDebounces(t *testing.T) {
	var searchHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/search") {
			searchHits.Add(1)
		}
		rpcOK(w, body["rpcId"], map[string]any{"items": []any{}, "hasMore": false})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)

	st.UpdateSessionSearch(context.Background(), "a")
	st.UpdateSessionSearch(context.Background(), "ab")
	st.UpdateSessionSearch(context.Background(), "abc")

	time.Sleep(sessionSearchDebounce + 200*time.Millisecond)

	if searchHits.Load() != 1 {
		t.Fatalf("debounced search calls = %d, want 1", searchHits.Load())
	}
	snap := st.Snapshot()
	if snap.SearchRPCQuery != "abc" {
		t.Fatalf("rpc query = %q", snap.SearchRPCQuery)
	}
}

func TestSearchSessionsFallbackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/session/search") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body["rpcId"],
				"result": map[string]any{
					"ok":    false,
					"error": map[string]any{"code": "fail", "message": "nope"},
				},
			})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.Attach(client, nil)

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if snap.SearchLoading || !snap.SearchUseLocal {
			return
		}
		select {
		case <-done:
		default:
			close(done)
		}
	})

	st.UpdateSessionSearch(context.Background(), "x")
	wait(t, done)

	snap := st.Snapshot()
	if snap.SearchErr == "" {
		t.Fatal("expected search error")
	}
	if !snap.SearchUseLocal {
		t.Fatal("expected local fallback")
	}
}

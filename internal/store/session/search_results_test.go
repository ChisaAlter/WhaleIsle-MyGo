package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestNativeMergedSessionSearchStoreFixture(t *testing.T) {
	type request struct {
		RPCID   any `json:"rpcId"`
		Payload struct {
			Args struct {
				Request map[string]any `json:"request"`
			} `json:"args"`
		} `json:"payload"`
	}
	requests := make(chan request, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		_ = json.NewDecoder(r.Body).Decode(&req)
		requests <- req
		if r.URL.Path != "/api/rpc/session/search" && r.URL.Path != "/rpc/session/search" {
			if len(r.URL.Path) < len("session/search") || r.URL.Path[len(r.URL.Path)-len("session/search"):] != "session/search" {
				t.Errorf("unexpected mutation/RPC: %s", r.URL.Path)
			}
		}
		if req.Payload.Args.Request["query"] == "failure" {
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": false, "error": map[string]any{"code": "fail", "message": "index unavailable"}}})
			return
		}
		rpcOK(w, req.RPCID, map[string]any{"items": []map[string]any{{"sessionId": "rank-b", "snippet": "first"}, {"sessionId": "rank-a", "snippet": "second"}}, "hasMore": true})
	}))
	defer srv.Close()
	c := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s := New()
	s.Attach(c, nil)
	idle := func() {
		t.Helper()
		end := time.Now().Add(3 * time.Second)
		for time.Now().Before(end) {
			if !s.Snapshot().SearchLoading {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("search did not settle")
	}
	s.UpdateSessionSearch(context.Background(), "n")
	s.UpdateSessionSearch(context.Background(), "ne")
	s.UpdateSessionSearch(context.Background(), " needle ")
	idle()
	req := <-requests
	if !reflect.DeepEqual(req.Payload.Args.Request, map[string]any{"query": "needle"}) {
		t.Fatalf("wire=%v", req.Payload.Args.Request)
	}
	select {
	case <-requests:
		t.Fatal("debounce dispatched extra query")
	default:
	}
	snap := s.Snapshot()
	if !snap.SearchHasMore || len(snap.SearchResults) != 2 || snap.SearchResults[0].SessionId != "rank-b" || snap.SearchResults[0].Snippet != "first" {
		t.Fatalf("lost search page: %+v", snap.SearchResults)
	}
	snap.SearchResults[0].Snippet = "mutated copy"
	if s.Snapshot().SearchResults[0].Snippet != "first" {
		t.Fatal("snapshot aliases store")
	}
	// Search is Host-global. Its ownership does not depend on main selection.
	// Change the field directly: Select also starts unrelated projection reads.
	s.mu.Lock()
	s.selectedID = "other"
	s.mu.Unlock()
	if s.Snapshot().SearchRPCQuery != "needle" {
		t.Fatal("selection discarded Host-global search")
	}
	s.UpdateSessionSearch(context.Background(), "failure")
	idle()
	<-requests
	if snap = s.Snapshot(); !snap.SearchUseLocal || snap.SearchErr == "" || len(snap.SearchResults) != 0 || snap.SearchHasMore {
		t.Fatal("failure did not expose clean local fallback")
	}
	// Late completion is rejected on query, clear, Host replacement and Detach.
	for _, change := range []string{"query", "clear", "host", "detach", "cancel", "events"} {
		t.Run(change, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			delayed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req request
				_ = json.NewDecoder(r.Body).Decode(&req)
				close(entered)
				<-release
				rpcOK(w, req.RPCID, map[string]any{"items": []map[string]any{{"sessionId": "stale", "snippet": "old Host"}}, "hasMore": true})
			}))
			defer delayed.Close()
			st := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: delayed.URL}}
			st.Attach(client, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			st.mu.Lock()
			st.searchInput = "same"
			st.searchLoading = true
			gen := st.searchSeq
			st.mu.Unlock()
			done := make(chan struct{})
			go func() { st.runSessionSearch(ctx, client, gen, "same"); close(done) }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("request did not start")
			}
			switch change {
			case "query":
				st.UpdateSessionSearch(context.Background(), "new")
			case "clear":
				st.UpdateSessionSearch(context.Background(), "")
			case "host":
				st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://unused.invalid"}}, nil)
			case "detach":
				st.Detach()
			case "cancel":
				cancel()
			case "events":
				st.BeginEvents(context.Background(), client)
			}
			close(release)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("late request did not finish")
			}
			if change == "events" {
				if len(st.Snapshot().SearchResults) != 1 || st.Snapshot().SearchLoading {
					t.Fatal("independent event reconnect blocked search")
				}
			} else if len(st.Snapshot().SearchResults) != 0 {
				t.Fatal("stale result published")
			}
			st.mu.Lock()
			st.clearSearchLocked()
			st.mu.Unlock()
		})
	}
	// Same text can be searched again after a Host replacement: no stale-input suppression.
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.UpdateSessionSearch(context.Background(), "failure")
	idle()
	<-requests
}

package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
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
	synctest.Test(t, func(t *testing.T) {
		type request struct {
			query string
			at    time.Time
			done  <-chan struct{}
		}
		requests := make(chan request, 16)
		notifications := make(chan struct{}, 1)
		releaseResponse := make(chan struct{})
		released := false
		release := func() {
			if !released {
				close(releaseResponse)
				released = true
			}
		}
		defer release()
		// Keep the real Client.Call envelope/worker, replacing only its HTTP
		// transport. No socket escapes the synctest bubble's scheduling barrier.
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://search-fixture.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/session/search" {
				t.Errorf("unexpected search RPC %s", r.URL.Path)
			}
			var body struct {
				RPCID   string `json:"rpcId"`
				Payload struct {
					Args struct {
						Request struct {
							Query string `json:"query"`
						} `json:"request"`
					} `json:"args"`
				} `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, err
			}
			select {
			case requests <- request{query: body.Payload.Args.Request.Query, at: time.Now(), done: r.Context().Done()}:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			select {
			case <-releaseResponse:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			response := httptest.NewRecorder()
			rpcOK(response, body.RPCID, map[string]any{"items": []any{}, "hasMore": false})
			return response.Result(), nil
		})}}
		st := New()
		st.Attach(client, func() {
			select {
			case notifications <- struct{}{}:
			default:
			}
		})
		defer st.Detach()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		assertNoRequest := func(phase string) {
			t.Helper()
			// Wait drains every runnable worker before sampling, so a delayed
			// goroutine cannot hide an obsolete RPC behind the assertion.
			synctest.Wait()
			select {
			case got := <-requests:
				t.Fatalf("%s: unexpected search query %q", phase, got.query)
			default:
			}
		}
		st.UpdateSessionSearch(ctx, "a")
		assertNoRequest("first input")
		time.Sleep(50 * time.Millisecond)
		st.UpdateSessionSearch(ctx, "ab")
		assertNoRequest("second input")
		time.Sleep(50 * time.Millisecond)
		st.UpdateSessionSearch(ctx, "abc")
		finalInputAt := time.Now()
		assertNoRequest("final input")
		time.Sleep(sessionSearchDebounce - time.Nanosecond)
		assertNoRequest("before final debounce deadline")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		var got request
		select {
		case got = <-requests:
		default:
			t.Fatal("final query did not reach the transport after its debounce deadline")
		}
		if got.query != "abc" || got.at.Sub(finalInputAt) != sessionSearchDebounce {
			t.Fatalf("request=%q after %v, want abc after %v", got.query, got.at.Sub(finalInputAt), sessionSearchDebounce)
		}
		if snap := st.Snapshot(); !snap.SearchLoading || snap.SearchRPCQuery != "" {
			t.Fatalf("blocked response was already published: loading=%v query=%q", snap.SearchLoading, snap.SearchRPCQuery)
		}
		time.Sleep(sessionSearchDebounce)
		assertNoRequest("while final response is pending")
		release()
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for {
			snap := st.Snapshot()
			if !snap.SearchLoading && snap.SearchRPCQuery == "abc" {
				break
			}
			select {
			case <-notifications:
			case <-deadline.C:
				t.Fatalf("search result was not published: loading=%v query=%q error=%q", snap.SearchLoading, snap.SearchRPCQuery, snap.SearchErr)
			}
		}
		assertNoRequest("after response publication")
		select {
		case <-got.done:
		default:
			t.Fatal("settled search did not release its request context")
		}
		st.UpdateSessionSearch(ctx, "abc") // unchanged input must not schedule again
		time.Sleep(sessionSearchDebounce)
		assertNoRequest("after another complete debounce window")
		st.UpdateSessionSearch(ctx, "abcd")
		st.UpdateSessionSearch(ctx, "") // clearing cancels the pending timer
		time.Sleep(sessionSearchDebounce)
		assertNoRequest("after clearing pending input")
		if snap := st.Snapshot(); snap.SearchLoading || snap.SearchRPCQuery != "" {
			t.Fatal("cleared search retained a loading/result state")
		}
	})
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

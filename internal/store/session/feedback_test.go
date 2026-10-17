package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func waitFeedbackDone(t *testing.T, st *Store) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st.Snapshot().FeedbackBusyID == "" && st.Snapshot().LastAction != "" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("feedback still busy: %+v", st.Snapshot())
}

func TestRateMessageCallsPutWhenPresent(t *testing.T) {
	var (
		mu        sync.Mutex
		gotRating string
		gotNote   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/messageFeedback/put" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		payload := body["payload"].(map[string]any)
		args := payload["args"].(map[string]any)
		req := args["request"].(map[string]any)
		mu.Lock()
		gotRating, _ = req["rating"].(string)
		gotNote, _ = req["note"].(string)
		mu.Unlock()
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

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {})
	st.mu.Lock()
	st.selectedID = "sess-1"
	st.mu.Unlock()

	st.RateMessage(context.Background(), "msg-1", MessageFeedbackUp, "")
	waitFeedbackDone(t, st)
	snap := st.Snapshot()
	mu.Lock()
	rating, note := gotRating, gotNote
	mu.Unlock()
	if rating != MessageFeedbackUp {
		t.Fatalf("rating sent = %q", rating)
	}
	if note != "" {
		t.Fatalf("note = %q", note)
	}
	if snap.LastAction != "feedback up saved" {
		t.Fatalf("lastAction = %q", snap.LastAction)
	}
	if snap.MessageRatings["msg-1"] != MessageFeedbackUp {
		t.Fatalf("MessageRatings = %#v", snap.MessageRatings)
	}
}

func TestRateMessageSendsNote(t *testing.T) {
	var gotNote string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/messageFeedback/put" {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		payload := body["payload"].(map[string]any)
		args := payload["args"].(map[string]any)
		req := args["request"].(map[string]any)
		gotNote, _ = req["note"].(string)
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

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {})
	st.mu.Lock()
	st.selectedID = "sess-1"
	st.mu.Unlock()

	st.RateMessage(context.Background(), "msg-1", MessageFeedbackDown, "too vague")
	waitFeedbackDone(t, st)
	if gotNote != "too vague" {
		t.Fatalf("note = %q", gotNote)
	}
	if st.Snapshot().MessageRatings["msg-1"] != MessageFeedbackDown {
		t.Fatalf("ratings = %#v", st.Snapshot().MessageRatings)
	}
}

func TestRateMessageLocalWhenEndpointMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {})
	st.mu.Lock()
	st.selectedID = "sess-1"
	st.mu.Unlock()

	st.RateMessage(context.Background(), "msg-1", MessageFeedbackDown, "")
	waitFeedbackDone(t, st)
	if got := st.Snapshot().LastAction; got != "feedback down (local)" {
		t.Fatalf("lastAction = %q", got)
	}
}

func TestRateMessageIgnoresWhileBusy(t *testing.T) {
	release := make(chan struct{})
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/messageFeedback/put" {
			http.NotFound(w, r)
			return
		}
		hits++
		<-release
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
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

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {})
	st.mu.Lock()
	st.selectedID = "sess-1"
	st.mu.Unlock()

	st.RateMessage(context.Background(), "msg-1", MessageFeedbackUp, "")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && st.Snapshot().FeedbackBusyID == "" {
		time.Sleep(2 * time.Millisecond)
	}
	if st.Snapshot().FeedbackBusyID != "msg-1" {
		t.Fatalf("expected busy msg-1, got %q", st.Snapshot().FeedbackBusyID)
	}
	st.RateMessage(context.Background(), "msg-2", MessageFeedbackDown, "")
	close(release)
	waitFeedbackDone(t, st)
	if hits != 1 {
		t.Fatalf("hits = %d want 1", hits)
	}
}

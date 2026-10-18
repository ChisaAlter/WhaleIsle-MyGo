package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"whaleisle/internal/dsh"
)

func TestReferenceCandidatesIdentityAndQuote(t *testing.T) {
	var sessionCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.HasSuffix(r.URL.Path, "/fileReferences/list") {
			rpcOK(w, body["rpcId"], []any{map[string]any{"kind": "file", "path": "my file.txt"}, map[string]any{"kind": "directory", "path": "folder one"}})
			return
		}
		sessionCalls.Add(1)
		rpcOK(w, body["rpcId"], []any{map[string]any{"sessionId": "a", "label": "same title", "mention": "@[same title](dsh-session:ImEi)"}, map[string]any{"sessionId": "b", "label": "same title", "mention": "@[same title](dsh-session:ImIi)"}, map[string]any{"sessionId": "c", "label": "unsafe fallback"}})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "s"
	got := s.ListReferenceCandidates(context.Background(), "s", "", false)
	if len(got) != 4 || got[2].Label != got[3].Label || got[2].SessionID == got[3].SessionID || got[2].Mention == got[3].Mention {
		t.Fatalf("%+v", got)
	}
	got = s.ListReferenceCandidates(context.Background(), "s", "my ", true)
	if len(got) != 2 || sessionCalls.Load() != 1 {
		t.Fatalf("quoted = %+v calls=%d", got, sessionCalls.Load())
	}
	if got = s.ListReferenceCandidates(context.Background(), "other", "", false); len(got) > 0 {
		t.Fatal("wrong selected owner")
	}
}
func TestReferenceCandidatesDiscardOldGeneration(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(entered)
		<-release
		rpcOK(w, body["rpcId"], []any{map[string]any{"kind": "file", "path": "old.txt"}})
	}))
	defer srv.Close()
	s := New()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.selectedID = "s"
	done := make(chan []ReferenceCandidate, 1)
	go func() { done <- s.ListReferenceCandidates(context.Background(), "s", "", true) }()
	<-entered
	s.mu.Lock()
	s.followGeneration++
	s.mu.Unlock()
	close(release)
	if got := <-done; len(got) > 0 {
		t.Fatalf("stale results %+v", got)
	}
}

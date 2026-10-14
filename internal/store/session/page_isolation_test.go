package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"whaleisle/internal/dsh"
)

func TestHistoryPageLateResultCannotMutateNewConversation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(started)
		<-release
		rpcOK(w, body["rpcId"], map[string]any{"hasMore": false, "records": []map[string]any{pageEventRecord(1, "user/message", "late A")}})
	}))
	defer srv.Close()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	s.Attach(client, nil)
	s.selectedID = "A"
	generation := s.followGeneration
	requestID := s.historyPageRequest
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		s.runHistoryPage(context.Background(), client, "A", generation, 90, 80, requestID)
	}()
	wait(t, started)
	// Use the same production selection invalidation, without unrelated RPCs.
	s.mu.Lock()
	s.selectSessionLocked("B")
	s.messages = []Message{{ID: "B", Body: "B", Seq: 7}}
	s.historyThroughSeq = 7
	s.historyHasMore = true
	s.historyPageBusy = true
	s.mu.Unlock()
	before := s.Snapshot()
	close(release)
	wait(t, finished)
	if got := s.Snapshot(); !reflect.DeepEqual(got, before) || s.historyThroughSeq != 7 {
		t.Fatalf("late history changed B: %+v", got)
	}
}

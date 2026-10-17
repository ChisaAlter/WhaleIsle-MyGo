package session

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

func TestPinSessionUpdatesPinnedIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/workspace/pinSession") {
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"pinnedSessionIds": []string{"a", "b"}})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.mu.Lock()
	st.items = []Item{
		{SessionID: "a", Title: "A"},
		{SessionID: "b", Title: "B"},
		{SessionID: "c", Title: "C"},
	}
	st.mu.Unlock()

	done := make(chan struct{})
	st.Attach(client, func() {
		snap := st.Snapshot()
		if len(snap.PinnedSessionIDs) == 2 && snap.Items[0].Pinned && !snap.Items[2].Pinned {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.PinSession(context.Background(), "a")
	wait(t, done)
}

func TestApplyWorkspacePinnedFrame(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.items = []Item{{SessionID: "p1", Title: "P"}}
	s.mu.Unlock()

	raw, _ := json.Marshal(map[string]any{
		"type":             "pinned",
		"pinnedSessionIds": []string{"p1"},
	})
	s.applyWorkspaceFollowFrame(raw)
	st := s.Snapshot()
	if len(st.PinnedSessionIDs) != 1 || !st.Items[0].Pinned {
		t.Fatalf("%+v", st)
	}
}

func TestUnpinSessionRPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/workspace/unpinSession") {
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{"pinnedSessionIds": []string{}})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st := New()
	st.setPinnedSessionIDs([]string{"x"})
	st.Attach(client, nil)
	st.UnpinSession(context.Background(), "x")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(st.Snapshot().PinnedSessionIDs) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected empty pinned list")
}

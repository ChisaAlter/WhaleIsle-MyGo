package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestNextPromptManagedPresentationListControlAndClear(t *testing.T) {
	s := New()
	s.selectedID = "a"
	managed := json.RawMessage(`{"sessionListMetadata":{"blank":false,"presentation":{"owner":"plugin","title":"Owned","composer":"managed"}}}`)
	ordinary := json.RawMessage(`{"sessionListMetadata":{"blank":false}}`)
	rows := []api.SessionListValueItemsItem{{SessionId: "a", AgentAvailable: true, Projections: &api.SessionListValueItemsItemProjections{Kind: "cached", AsOfSeq: 1, Values: managed}}, {SessionId: "b", AgentAvailable: true}}
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://presentation.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/session/list" {
			return nil, fmt.Errorf("unexpected method %s", r.URL.Path)
		}
		return composerRegressionReply(r, api.SessionListValue{Items: rows})
	})}}
	refresh := func() {
		s.mu.Lock()
		client, generation := s.beginListRefreshLocked()
		s.mu.Unlock()
		s.runRefreshList(context.Background(), client, generation)
	}
	assertManaged := func(id string, want bool) {
		t.Helper()
		for _, item := range s.Snapshot().Items {
			if item.SessionID == id {
				got := item.Presentation != nil && item.Presentation.Composer == "managed"
				if got != want {
					t.Fatalf("%s managed=%v, want %v: %+v", id, got, want, item.Presentation)
				}
				return
			}
		}
		t.Fatalf("missing Session %s", id)
	}
	refresh()
	assertManaged("a", true)
	// This is the source's sessionListMetadata projection, not a guessed field
	// on session/list itself. Its ordinary value explicitly withdraws managed.
	s.applySessionPresentationControlLocked(json.RawMessage(`{"type":"projection","sessionId":"a","key":"sessionListMetadata","seq":5,"value":{"blank":false}}`))
	assertManaged("a", false)
	s.applySessionPresentationControlLocked(json.RawMessage(`{"type":"projection","sessionId":"b","key":"sessionListMetadata","seq":6,"value":{"presentation":{"owner":"plugin","title":"B","composer":"managed"}}}`))
	assertManaged("b", true)
	assertManaged("a", false)
	// A cached watermark is not comparable to the connected sequence. Even a
	// numerically larger cold hint cannot resurrect an already-cleared owner.
	rows[0].Projections.AsOfSeq = 9999
	refresh()
	assertManaged("a", false)
	// Availability-only added summaries must not withdraw known presentation.
	owner := s.BeginEvents(context.Background(), s.client)
	s.OnEmit(owner, dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/added", Args: []json.RawMessage{json.RawMessage(`{"sessionId":"b","agentAvailable":true,"running":false,"blank":false,"updatedAt":7}`)}})
	assertManaged("b", true)
	s.applySessionPresentationControlLocked(json.RawMessage(`{"type":"baseline","value":{"projections":{"a":{"asOfSeq":8,"values":{"sessionListMetadata":{"presentation":{"owner":"plugin","title":"A","composer":"managed"}}}}}}}`))
	assertManaged("a", true)
	s.applySessionPresentationValuesLocked("a", "sequenced", 9, ordinary)
	assertManaged("a", false)
	s.applySessionPresentationValuesLocked("a", "sequenced", 8, managed)
	assertManaged("a", false)
	// The durable presentation event supports explicit clearing, independently
	// of list/control timing, just as the original presentation fold does.
	s.applyPersistenceEvent(json.RawMessage(`{"type":"session/presentation","seq":10,"data":{"owner":"plugin","title":"A","composer":"managed"}}`))
	assertManaged("a", true)
	s.applyPersistenceEvent(json.RawMessage(`{"type":"session/presentation","seq":11,"data":null}`))
	assertManaged("a", false)
	s.Attach(&dsh.Client{}, nil)
	assertManaged("b", false)
}

func TestNextPromptReopenedSameIDRejectsOldSuggestion(t *testing.T) {
	s := New()
	s.selectedID = "a"
	s.historyReady = true
	s.items = []Item{{SessionID: "a"}, {SessionID: "b"}}
	published := make(chan struct{}, 4)
	s.notify = func() { published <- struct{}{} }
	var calls atomic.Int32
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://next-prompt.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/conversation.next-prompt" {
			return nil, fmt.Errorf("unexpected method %s", r.URL.Path)
		}
		n := calls.Add(1)
		return composerRegressionReply(r, map[string]any{"turn": 1, "seq": n, "text": fmt.Sprintf("next-%d", n)})
	})}}
	fetch := func() NextPromptSuggestion {
		t.Helper()
		s.SyncNextPrompt(context.Background(), true)
		select {
		case <-published:
		case <-time.After(time.Second):
			t.Fatal("completed suggestion was not published")
		}
		return s.SyncNextPrompt(context.Background(), true)
	}
	first := fetch()
	if first.Text != "next-1" {
		t.Fatalf("first suggestion %+v", first)
	}
	s.mu.Lock()
	s.selectSessionLocked("b")
	s.selectSessionLocked("a")
	s.historyReady, s.historyLoading = true, false
	s.mu.Unlock()
	if got := s.AcceptNextPrompt(first); got != "" {
		t.Fatalf("old open generation supplied the reopened composer: %q", got)
	}
	second := fetch()
	if second.Text != "next-2" || calls.Load() != 2 {
		t.Fatalf("same ID reused old generation: %+v, requests=%d", second, calls.Load())
	}
	s.applySessionPresentationControlLocked(json.RawMessage(`{"type":"projection","sessionId":"a","key":"sessionListMetadata","seq":20,"value":{"presentation":{"owner":"plugin","title":"A","composer":"managed"}}}`))
	if got := s.AcceptNextPrompt(second); got != "" {
		t.Fatal("managed transition accepted a retained ordinary suggestion")
	}
	if got := s.SyncNextPrompt(context.Background(), true); got.Text != "" || calls.Load() != 2 {
		t.Fatal("managed composer fetched or displayed a suggestion")
	}
	s.applySessionPresentationControlLocked(json.RawMessage(`{"type":"projection","sessionId":"a","key":"sessionListMetadata","seq":21,"value":{"presentation":null}}`))
	third := fetch()
	if third.Text != "next-3" || s.AcceptNextPrompt(third) != "next-3" || s.AcceptNextPrompt(third) != "" {
		t.Fatal("cleared managed owner did not get a fresh, single-use suggestion")
	}
}

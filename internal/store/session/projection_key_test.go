package session

import (
	"encoding/json"
	"testing"
)

func TestProjectionKeyNamesTheStoreRevision(t *testing.T) {
	s := New()
	first := s.Snapshot().ProjectionKey()
	if !first.Valid() {
		t.Fatal("a projected view has no key")
	}
	frame := s.BeginFrame()
	if got := frame.Snapshot().ProjectionKey(); got != first {
		t.Fatalf("frame view of the same revision: %+v, want %+v", got, first)
	}
	frame.EndFrame()
	if again := s.Snapshot().ProjectionKey(); again != first {
		t.Fatalf("unchanged revision changed key: %+v -> %+v", first, again)
	}
	s.SetLastAction("changed")
	if next := s.Snapshot().ProjectionKey(); next == first || !next.Valid() {
		t.Fatalf("a new revision kept key %+v", next)
	}
	if other := New().Snapshot().ProjectionKey(); other == first {
		t.Fatal("two stores share a key")
	}
	if (ViewState{}).ProjectionKey().Valid() {
		t.Fatal("a hand-built view claims a store revision")
	}
}

func TestRosterProjectionKeyStableDuringStreaming(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.items = []Item{{SessionID: "s1", Title: "Session 1", Cwd: "/ws"}}
	s.selectedID = "s1"
	s.invalidateViewLocked()
	s.mu.Unlock()

	snap1 := s.Snapshot()
	rosterKey1 := snap1.RosterProjectionKey()
	histKey1 := snap1.HistoryProjectionKey()
	if !rosterKey1.Valid() || !histKey1.Valid() {
		t.Fatal("expected valid projection keys")
	}

	// Simulate streaming message chunk: history changes, roster does not.
	s.mu.Lock()
	s.messages = append(s.messages, Message{ID: "stream-1", Kind: MessageKindText, Role: "assistant", Body: "hello", Streaming: true})
	s.invalidateStreamViewLocked()
	s.mu.Unlock()

	snap2 := s.Snapshot()
	if snap2.RosterProjectionKey() != rosterKey1 {
		t.Fatal("streaming message chunk invalidated RosterProjectionKey")
	}
	if snap2.HistoryProjectionKey() == histKey1 {
		t.Fatal("streaming message chunk did not advance HistoryProjectionKey")
	}

	// Simulate roster change: RosterProjectionKey advances, HistoryProjectionKey stays stable.
	histKey2 := snap2.HistoryProjectionKey()
	s.mu.Lock()
	s.items = append(s.items, Item{SessionID: "s2", Title: "Session 2", Cwd: "/ws"})
	s.invalidateViewLocked()
	s.mu.Unlock()

	snap3 := s.Snapshot()
	if snap3.RosterProjectionKey() == rosterKey1 {
		t.Fatal("roster update kept stale RosterProjectionKey")
	}
	if snap3.HistoryProjectionKey() != histKey2 {
		t.Fatal("roster update invalidated HistoryProjectionKey")
	}
}

func TestRosterKeyAndDurableCacheSurviveFollowStreamAndQueuedPending(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.items = []Item{{SessionID: "s1", Title: "Session 1", Cwd: "/ws"}}
	s.selectedID = "s1"
	s.messages = []Message{
		{ID: "u1", Seq: 1, Kind: MessageKindText, Role: "user", Body: "hi", EventType: "user/message", Raw: json.RawMessage(`{"type":"user/message","seq":1,"data":{"turn":1}}`)},
	}
	target := &ComposerTarget{
		SessionID: "s1",
		store:     s,
		client:    s.client,
		epoch:     s.composerHostGeneration,
	}
	s.composerPending = map[string]*composerPending{
		"req-1": {
			PendingSubmission: PendingSubmission{
				RequestID: "req-1",
				SessionID: "s1",
				Text:      "queued follow-up",
				Placement: "queued",
				Accepted:  true,
				Order:     1,
			},
			target: target,
		},
	}
	s.invalidateViewLocked()
	s.mu.Unlock()

	snap0 := s.Snapshot()
	rosterKey0 := snap0.RosterProjectionKey()
	histKey0 := snap0.HistoryProjectionKey()

	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"start","attemptId":"a1","revision":1,"startedAfterSeq":1,"turn":1,"step":1}}`))
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":2,"index":0,"time":1000,"chunk":{"type":"block-start","index":0,"blockType":"text"}}}`))
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":3,"index":1,"time":1010,"chunk":{"type":"text-delta","index":0,"text":"streamed"}}}`))

	snap1 := s.Snapshot()
	if snap1.RosterProjectionKey() != rosterKey0 {
		t.Fatal("follow assistant-stream frames invalidated RosterProjectionKey")
	}
	if snap1.HistoryProjectionKey() == histKey0 {
		t.Fatal("follow assistant-stream frames did not advance HistoryProjectionKey")
	}
	s.mu.Lock()
	durableValid := s.durableCacheValid
	s.mu.Unlock()
	if !durableValid {
		t.Fatal("queued composerPending disabled durableTimedCache during streaming")
	}
}

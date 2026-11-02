package session

import (
	"encoding/json"
	"testing"
)

func TestRenderSnapshotSharesOneCloneUntilTheRevisionMoves(t *testing.T) {
	s := New()
	s.mu.Lock()
	s.selectedID = "s1"
	s.items = []Item{{SessionID: "s1", Title: "Session 1"}}
	s.messages = []Message{{ID: "u1", Seq: 1, Kind: MessageKindText, Role: "user", Body: "hi", EventType: "user/message", Raw: json.RawMessage(`{"type":"user/message","seq":1,"data":{"turn":1}}`)}}
	s.invalidateViewLocked()
	s.mu.Unlock()

	first := s.RenderSnapshot()
	second := s.RenderSnapshot()
	if len(first.Messages) != 1 || &first.Messages[0] != &second.Messages[0] {
		t.Fatal("an unchanged revision was cloned again")
	}
	if deep := s.Snapshot(); &deep.Messages[0] == &first.Messages[0] {
		t.Fatal("Snapshot shares its messages with the render view")
	}
	if first.ProjectionKey() != s.Snapshot().ProjectionKey() || !first.ProjectionKey().Valid() {
		t.Fatal("the render view lost the store revision's projection key")
	}

	s.SetLastAction("moved")
	third := s.RenderSnapshot()
	if third.LastAction != "moved" || third.ProjectionKey() == first.ProjectionKey() {
		t.Fatalf("a new revision was not served: %q", third.LastAction)
	}
	if first.LastAction == "moved" {
		t.Fatal("a view handed out earlier changed under its holder")
	}
}

func TestRenderSnapshotIsSafeForConcurrentReaders(t *testing.T) {
	s := New()
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 200; j++ {
				_ = s.RenderSnapshot()
				s.SetLastAction("tick")
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

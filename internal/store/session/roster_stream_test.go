package session

import (
	"context"
	"encoding/json"
	"testing"
)

// These tests drive followed frames through the entry points the follow loop
// uses (applyFollowFrameForTarget, applyFollowFrame), which publish the
// revision with fire. The *Locked variants skip that step, so a test built on
// them cannot see the roster being marked dirty by publication.

const (
	streamStartFrame = `{"type":"assistant-stream","frame":{"type":"start","attemptId":"a1","revision":1,"startedAfterSeq":1,"turn":1,"step":1}}`
	streamBlockFrame = `{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":2,"index":0,"time":1000,"chunk":{"type":"block-start","index":0,"blockType":"text"}}}`
	streamDeltaFrame = `{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":3,"index":1,"time":1010,"chunk":{"type":"text-delta","index":0,"text":"streamed"}}}`
	streamMoreFrame  = `{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":4,"index":2,"time":1020,"chunk":{"type":"text-delta","index":0,"text":" more"}}}`
)

func newStreamingStore(t *testing.T) (*Store, *int) {
	t.Helper()
	s := New()
	notified := new(int)
	s.mu.Lock()
	s.items = []Item{{SessionID: "s1", Title: "Session 1", Cwd: "/ws"}}
	s.selectedID = "s1"
	s.messages = []Message{
		{ID: "u1", Seq: 1, Kind: MessageKindText, Role: "user", Body: "hi", EventType: "user/message", Raw: json.RawMessage(`{"type":"user/message","seq":1,"data":{"turn":1}}`)},
	}
	s.notify = func() { *notified++ }
	s.invalidateViewLocked()
	s.mu.Unlock()
	return s, notified
}

func TestRosterKeyStableAcrossStreamChunksPublishedByTheFollowLoop(t *testing.T) {
	for _, entry := range []string{"applyFollowFrame", "applyFollowFrameForTarget"} {
		t.Run(entry, func(t *testing.T) {
			s, notified := newStreamingStore(t)
			apply := func(frame string) {
				if entry == "applyFollowFrame" {
					s.applyFollowFrame(json.RawMessage(frame))
					return
				}
				s.mu.Lock()
				generation := s.followGeneration
				s.mu.Unlock()
				if !s.applyFollowFrameForTarget(context.Background(), "s1", generation, json.RawMessage(frame)) {
					t.Fatalf("frame was not applied to the followed session: %s", frame)
				}
			}

			before := s.Snapshot()
			roster, history := before.RosterProjectionKey(), before.HistoryProjectionKey()
			*notified = 0

			for i, frame := range []string{streamStartFrame, streamBlockFrame, streamDeltaFrame, streamMoreFrame} {
				apply(frame)
				snap := s.Snapshot()
				if snap.RosterProjectionKey() != roster {
					t.Fatalf("frame %d invalidated RosterProjectionKey", i)
				}
				if snap.HistoryProjectionKey() == history {
					t.Fatalf("frame %d left HistoryProjectionKey on a stale revision", i)
				}
			}
			if *notified == 0 {
				t.Fatal("stream chunks published nothing to the UI")
			}
			if got := len(s.Snapshot().Messages); got != 2 {
				t.Fatalf("streamed reply is not in the view: %d messages", got)
			}
		})
	}
}

func TestRosterKeyAdvancesForFramesThatAreNotStreamChunks(t *testing.T) {
	s, _ := newStreamingStore(t)
	roster := s.Snapshot().RosterProjectionKey()

	// A durable event can retitle, reorder or finish a session, so it must reach
	// every roster memo even when it arrives mid-stream.
	s.applyFollowFrame(json.RawMessage(streamStartFrame))
	s.applyFollowFrame(json.RawMessage(streamBlockFrame))
	if s.Snapshot().RosterProjectionKey() != roster {
		t.Fatal("stream frames moved the roster")
	}
	s.applyFollowFrame(json.RawMessage(`{"type":"event","event":{"type":"user/message","seq":2,"data":{"turn":2}}}`))
	if s.Snapshot().RosterProjectionKey() == roster {
		t.Fatal("a durable event left the roster revision unchanged")
	}
}

func TestRosterKeyAdvancesWhenAStreamRebaselines(t *testing.T) {
	s, _ := newStreamingStore(t)
	roster := s.Snapshot().RosterProjectionKey()

	// Revision 9 breaks the dense sequence: the attempt must be rebaselined, which
	// restarts the follow and is not a plain chunk.
	s.applyFollowFrame(json.RawMessage(`{"type":"assistant-stream","frame":{"type":"chunk","attemptId":"a1","revision":9,"index":0,"time":1000,"chunk":{"type":"text-delta","index":0,"text":"x"}}}`))
	if s.Snapshot().RosterProjectionKey() == roster {
		t.Fatal("a rebaseline left the roster revision unchanged")
	}
}

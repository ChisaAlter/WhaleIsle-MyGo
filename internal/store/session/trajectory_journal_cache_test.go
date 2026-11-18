package session

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTrajectoryStreamingCachePreservesJournalAndEarlierViews(t *testing.T) {
	s := benchmarkStore(60)
	live := streamingRow()
	s.messages = append(s.messages, live)
	firstToken := float64(1234)
	a := &assistantStreamAttempt{AttemptID: "live", Turn: 1, Step: 2, StartedAfterSeq: 60,
		rows: map[int]string{0: live.ID}, trajectoryFirstToken: &firstToken}
	s.assistantStream.active = a
	frame := s.BeginFrame()
	defer frame.EndFrame()
	check := func() ViewState {
		t.Helper()
		view := frame.Snapshot()
		s.mu.Lock()
		fullJournal := s.trajectoryJournalMessagesLocked()
		s.mu.Unlock()
		if !reflect.DeepEqual(view.TrajectoryMessages, fullJournal) {
			t.Fatalf("incremental journal changed full-ledger semantics\ngot: %+v\nwant: %+v", view.TrajectoryMessages, fullJournal)
		}
		return view
	}
	old := check()
	frozen := cloneViewState(old)
	cache := s.trajectoryMessageCache
	s.mu.Lock()
	s.messages[len(s.messages)-1].Body += " next 中文"
	foldToolPreparingChunk(&a.tools, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"pending","name":"read","argumentsDelta":"{}"}`), 1235)
	s.invalidateViewLocked()
	s.mu.Unlock()
	current := check()
	if s.trajectoryMessageCache != cache {
		t.Fatal("stream-only change rebuilt durable journal")
	}
	if !reflect.DeepEqual(old, frozen) {
		t.Fatal("stream update mutated the previously published view")
	}
	if current.TrajectoryMessages[len(current.TrajectoryMessages)-1].Tool == nil {
		t.Fatal("new preparing tool missing from the journal")
	}

	// A same-size edit of an old source must invalidate even though the latest
	// stream row and event counts have not changed.
	s.mu.Lock()
	s.messages[0].Body = "changed old row"
	s.invalidateViewLocked()
	s.mu.Unlock()
	check()
	if s.trajectoryMessageCache == cache {
		t.Fatal("old source edit retained the durable journal")
	}
	for _, retained := range []bool{true, false} {
		s.mu.Lock()
		s.assistantStream.active = nil
		if retained {
			s.assistantStream.retained = a
		} else {
			s.assistantStream.retained = nil
			s.messages = s.messages[:len(s.messages)-1]
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		check()
	}
}

func TestFirstFrameClaimsFrozenArraysWithoutSharingAcrossHandles(t *testing.T) {
	s := benchmarkStore(30)
	a, b := s.BeginFrame(), s.BeginFrame()
	defer a.EndFrame()
	defer b.EndFrame()
	av := a.Snapshot()
	bv := b.Snapshot()
	if &av.Messages[0] != &s.viewCache.Messages[0] || &av.TrajectoryMessages[0] != &s.viewCache.TrajectoryMessages[0] {
		t.Fatal("first read-only pass cloned the frozen revision")
	}
	if &av.Messages[0] == &bv.Messages[0] || &av.TrajectoryMessages[0] == &bv.TrajectoryMessages[0] {
		t.Fatal("overlapping handles share top-level message arrays")
	}
	original := cloneViewState(av)
	s.SetLastAction("new revision")
	current := a.Snapshot()
	if &current.Messages[0] == &av.Messages[0] || &current.TrajectoryMessages[0] == &bv.TrajectoryMessages[0] {
		t.Fatal("state-only revision reclaimed arrays from an earlier pass")
	}
	if !reflect.DeepEqual(av, original) || !reflect.DeepEqual(bv, original) {
		t.Fatal("next revision changed the prior pass")
	}
	public := s.Snapshot()
	public.TrajectoryMessages[0].Raw[0] ^= 1
	if string(av.TrajectoryMessages[0].Raw) != string(original.TrajectoryMessages[0].Raw) {
		t.Fatal("public Snapshot modified frozen frame data")
	}
}

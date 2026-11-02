package session

import (
	"encoding/json"
	"testing"
)

// A frame reads SessionBatch, re-observes the roster and re-applies the Host's
// question faces every refresh. Those reads must not advance the Snapshot
// revision when nothing changed: doing so rebuilt the whole view on the next
// frame, which tmp/stalls captured as the per-frame stall on a long session.
func TestRepeatedObservationDoesNotAdvanceViewRevision(t *testing.T) {
	s := New()
	items := []Item{{SessionID: "a", Running: true}, {SessionID: "b"}}
	s.reconcileSidebarStatusesLocked(items)
	rows := s.viewRev
	s.reconcileSidebarStatusesLocked(items)
	s.reconcileSidebarStatusesLocked(items)
	if s.viewRev != rows {
		t.Fatalf("unchanged roster refresh advanced the revision: %d -> %d", rows, s.viewRev)
	}

	s.ensureBatchLocked()
	batch := s.viewRev
	s.ensureBatchLocked()
	s.ensureBatchLocked()
	if s.viewRev != batch {
		t.Fatalf("reading the batch advanced the revision: %d -> %d", batch, s.viewRev)
	}

	face := json.RawMessage(`{"active":[],"settled":[]}`)
	s.applyQuestionFaceLocked("a", "userQuestions", face, 7)
	question := s.viewRev
	// The Host resends the same face with a newer as-of sequence on each list
	// refresh; only the stored sequence moves.
	s.applyQuestionFaceLocked("a", "userQuestions", face, 8)
	s.applyQuestionFaceLocked("a", "userQuestions", face, 8)
	if s.viewRev != question {
		t.Fatalf("unchanged question face advanced the revision: %d -> %d", question, s.viewRev)
	}
	if got := s.questionProjections["a"].QuestionsSeq; got != 8 {
		t.Fatalf("newer as-of sequence not recorded: %v", got)
	}

	// A new face still advances it.
	s.applyQuestionFaceLocked("a", "userQuestions", json.RawMessage(`{"active":[],"settled":[]}`+" "), 9)
	if s.viewRev == question {
		t.Fatal("a changed question face did not advance the revision")
	}

	// Clearing a next-prompt that is not running changes nothing.
	next := s.viewRev
	s.clearNextPromptLocked()
	if s.viewRev != next {
		t.Fatalf("clearing an empty next-prompt advanced the revision: %d -> %d", next, s.viewRev)
	}
}

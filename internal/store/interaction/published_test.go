package interaction

import (
	"errors"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
)

func publishedFixture() *Store {
	s := New()
	s.mu.Lock()
	s.pending = []Pending{
		{AgentID: "A", EventID: "approval", Event: EventApproval},
		{AgentID: "A", EventID: "question", Event: EventUserQuestions},
	}
	s.waiters["approval"] = make(chan dsh.EventResultOutcome, 1)
	s.unlockPublish()
	return s
}

func TestInteractionSnapshotSharedUntilWrite(t *testing.T) {
	s := publishedFixture()
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	// Questions outrank approvals; the order is computed once per publish.
	if p := first.Value.Pending; len(p) != 2 || p[0].EventID != "question" {
		t.Fatalf("published order: %+v", p)
	}
	// Read-only sections and the live draft channel publish nothing.
	s.QuestionsCanSubmit("question")
	s.QuestionCards()
	s.SnapshotForAgent("A")
	if s.Snapshot() != first {
		t.Fatal("a section that changed nothing published a new version")
	}
}

func TestInteractionWritePublishesAndSignals(t *testing.T) {
	s := publishedFixture()
	before := s.Snapshot()
	changed := s.Changed()
	s.SetAutoAllow(true)
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 || !after.Value.AutoAllow {
		t.Fatalf("SetAutoAllow did not publish Rev+1: %d -> %d", before.Rev, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
}

// Pending entries are edited in place through findPending; a version a
// reader holds keeps its values.
func TestInteractionPendingEditLeavesHeldVersion(t *testing.T) {
	s := publishedFixture()
	held := s.Snapshot()
	s.ResolveAllow("approval")
	for _, p := range held.Value.Pending {
		if p.Sending {
			t.Fatal("in-place pending edit reached a published version")
		}
	}
	sending := false
	for _, p := range s.Snapshot().Value.Pending {
		sending = sending || (p.EventID == "approval" && p.Sending)
	}
	if !sending {
		t.Fatal("pending edit was not published")
	}
}

func TestInteractionDeliveryFaultMapping(t *testing.T) {
	s := publishedFixture()
	s.ResolveAllow("approval")
	s.OnResult(dsh.EventResult{EventID: "approval"}, errors.New("dial tcp 127.0.0.1:1: connectex: connection refused"))
	var f = s.Snapshot().Value.Pending
	for _, p := range f {
		if p.EventID == "approval" {
			if p.Error == nil || p.Error.Cause != errcopy.CauseNetwork || p.Error.SentenceText() == "" {
				t.Fatalf("delivery failure not classified: %+v", p.Error)
			}
			return
		}
	}
	t.Fatal("failed delivery dropped the card")
}

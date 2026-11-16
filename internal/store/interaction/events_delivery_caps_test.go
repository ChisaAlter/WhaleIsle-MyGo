package interaction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/dsh"
)

func TestInteractionDetachClearsAndCapsCancelledMap(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	mux := &dsh.Mux{}
	s.AttachClient(client, mux)

	for i := 0; i < maxCancelledEventIDs+50; i++ {
		s.OnCancel(dsh.EventsCancel{Type: dsh.EventCancel, EventID: fmt.Sprintf("evt-%d", i)})
	}
	s.mu.Lock()
	if got := len(s.cancelled); got > maxCancelledEventIDs {
		s.mu.Unlock()
		t.Fatalf("cancelled map exceeded cap: got=%d cap=%d", got, maxCancelledEventIDs)
	}
	if s.cancelled["evt-0"] {
		s.mu.Unlock()
		t.Fatal("oldest cancelled entry was not evicted")
	}
	if !s.cancelled[fmt.Sprintf("evt-%d", maxCancelledEventIDs+49)] {
		s.mu.Unlock()
		t.Fatal("newest cancelled entry missing")
	}
	s.mu.Unlock()

	s.Detach()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cancelled) != 0 || len(s.cancelledOrder) != 0 {
		t.Fatalf("Detach leaked cancelled entries: map=%d order=%d", len(s.cancelled), len(s.cancelledOrder))
	}
}

func TestBeginEventsDeliveryResetsOrphanSendingApprovalWithoutTouchingContinued(t *testing.T) {
	s := New()
	client := &dsh.Client{}
	mux := &dsh.Mux{}
	s.AttachClient(client, mux)

	old := s.BeginEventsDelivery(context.Background(), client, mux)
	if old == nil {
		t.Fatal("expected initial owner")
	}

	wf := dsh.EventsWaterfall{
		Type:    dsh.EventWaterfall,
		Event:   EventApproval,
		EventID: "orphan-approval",
		AgentID: "A",
		Request: json.RawMessage(`{"toolName":"bash"}`),
	}
	resultCh := make(chan *dsh.EventResultOutcome, 1)
	go func() {
		out, _ := s.HandleWaterfallForDelivery(old, wf)
		resultCh <- out
	}()
	awaitQuestionFixture(t, func() bool { return len(s.Snapshot().Pending) == 1 })

	// Start in-flight approval send, then drop carrier / begin new delivery generation
	// while the approval remains pending in-memory.
	s.ResolveAllow("orphan-approval")
	<-resultCh
	s.mu.Lock()
	s.pending = append(s.pending,
		Pending{EventID: "orphan-approval", Event: EventApproval, Sending: true},
		Pending{EventID: "continued-q", Event: EventUserQuestions, Continued: true, Sending: true},
	)
	s.mu.Unlock()

	fresh := s.BeginEventsDelivery(context.Background(), client, mux)
	if fresh == nil {
		t.Fatal("expected replacement owner")
	}
	defer s.RetireEventsDelivery(fresh)

	s.mu.Lock()
	defer s.mu.Unlock()
	var orphanSending, continuedSending bool
	for _, p := range s.pending {
		switch p.EventID {
		case "orphan-approval":
			orphanSending = p.Sending
		case "continued-q":
			continuedSending = p.Sending
		}
	}
	if orphanSending {
		t.Fatal("BeginEventsDelivery left non-continued Sending approval stuck")
	}
	if !continuedSending {
		t.Fatal("BeginEventsDelivery disrupted continued question Sending state")
	}
}

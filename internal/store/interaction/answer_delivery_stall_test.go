package interaction

import (
	"context"
	"testing"

	"whaleisle/internal/dsh"
)

// A retired delivery generation leaves nothing able to finish an outstanding
// send, so the card has to report that instead of claiming progress forever.
func TestRetiredDeliveryGenerationReleasesSendingCard(t *testing.T) {
	store := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	store.AttachClient(client, mux)
	owner := store.BeginEventsDelivery(context.Background(), client, mux)
	if owner == nil {
		t.Fatalf("delivery generation did not start")
	}
	store.mu.Lock()
	store.pending = append(store.pending, Pending{Event: EventUserQuestions, EventID: "event-b", AgentID: "A", Sending: true})
	store.mu.Unlock()

	store.RetireEventsDelivery(owner)

	pending := store.SnapshotForAgent("A").Pending
	if len(pending) != 1 || pending[0].Sending || pending[0].Error != QuestionDeliveryStopped {
		t.Fatalf("retired generation left the card in progress: %+v", pending)
	}
}

// A card whose attempt is still in flight belongs to that attempt: the sweep must
// not clear it, or a slow but healthy send would be reported as failed.
func TestLiveAttemptKeepsCardSending(t *testing.T) {
	store := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	store.AttachClient(client, mux)
	owner := store.BeginEventsDelivery(context.Background(), client, mux)
	store.mu.Lock()
	store.pending = append(store.pending, Pending{Event: EventUserQuestions, EventID: "event-b", AgentID: "A", Sending: true})
	store.questionRecovery.inFlight = map[string]questionAnswerAttempt{"event-b": {token: 1}}
	store.mu.Unlock()

	store.RetireEventsDelivery(owner)

	pending := store.SnapshotForAgent("A").Pending
	if len(pending) != 1 || !pending[0].Sending || pending[0].Error != "" {
		t.Fatalf("a live attempt was cleared: %+v", pending)
	}
}

// A card whose attempt vanished must not keep claiming progress, while a card
// that another attempt owns must be left alone.
func TestStalledAnswerAttemptReleasesOnlyAbandonedCard(t *testing.T) {
	store := New()
	store.mu.Lock()
	store.pending = append(store.pending,
		Pending{Event: EventUserQuestions, EventID: "abandoned", Continued: true, Sending: true},
		Pending{Event: EventUserQuestions, EventID: "accounted", Continued: true, Sending: true},
	)
	store.questionRecovery.inFlight = map[string]questionAnswerAttempt{"accounted": {token: 1}}
	store.mu.Unlock()

	store.mu.Lock()
	store.releaseStalledAnswerLocked("abandoned")
	store.releaseStalledAnswerLocked("accounted")
	store.mu.Unlock()

	byEvent := map[string]Pending{}
	for _, p := range store.Snapshot().Pending {
		byEvent[p.EventID] = p
	}
	if got := byEvent["abandoned"]; got.Sending || got.Error != QuestionDeliveryStopped {
		t.Fatalf("abandoned card kept progress: %+v", got)
	}
	if got := byEvent["accounted"]; !got.Sending || got.Error != "" {
		t.Fatalf("live attempt was disturbed: %+v", got)
	}
}

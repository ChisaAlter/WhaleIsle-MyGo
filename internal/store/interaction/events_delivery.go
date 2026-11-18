package interaction

import (
	"context"

	"whaleisle/internal/dsh"
)

// EventsDeliveryOwner is one answering $events attempt, independently of the
// retained Host client and durable question drafts. Gateway replays the same
// eventId to a new attempt after its previous client delivery is withdrawn.
type EventsDeliveryOwner struct {
	store   *Store
	client  *dsh.Client
	mux     *dsh.Mux
	ctx     context.Context
	cancel  context.CancelFunc
	retired bool
}

type eventsDelivery struct {
	owner          *EventsDeliveryOwner
	waiter         chan dsh.EventResultOutcome
	ctx            context.Context
	cancel         context.CancelFunc
	response       *dsh.EventDelivery
	scope          *dsh.EventDelivery
	responseCancel context.CancelFunc
	responseStop   func() bool
}

func (s *Store) eventsOwnerCurrentLocked(owner *EventsDeliveryOwner) bool {
	return owner != nil && owner.store == s && s.eventsDeliveryOwner == owner && !owner.retired && owner.ctx.Err() == nil && s.client == owner.client && s.mux == owner.mux
}

// BeginEventsDelivery replaces the old logical delivery before the new replay
// can enter. An old pump's later Retire cannot remove the new owner or waiter.
func (s *Store) BeginEventsDelivery(ctx context.Context, client *dsh.Client, mux *dsh.Mux) *EventsDeliveryOwner {
	if ctx == nil || ctx.Err() != nil || client == nil || mux == nil {
		return nil
	}
	s.mu.Lock()
	if s.client != client || s.mux != mux || ctx.Err() != nil {
		s.mu.Unlock()
		return nil
	}
	s.retireEventsDeliveryLocked(s.eventsDeliveryOwner)
	for i := range s.pending {
		if !s.pending[i].Continued && s.pending[i].Sending {
			s.pending[i].Sending = false
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	owner := &EventsDeliveryOwner{store: s, client: client, mux: mux, ctx: ctx, cancel: cancel}
	s.eventsDeliveryOwner = owner
	s.mu.Unlock()
	s.persistQuestionDrafts()
	s.fire()
	return owner
}

func (s *Store) registerEventsDeliveryLocked(owner *EventsDeliveryOwner, id string, waiter chan dsh.EventResultOutcome) *eventsDelivery {
	if prev := s.eventsDeliveries[id]; prev != nil {
		prev.cancel()
		if prev.responseCancel != nil {
			prev.responseCancel()
		}
		if prev.responseStop != nil {
			prev.responseStop()
		}
		if prev.waiter != nil && prev.waiter != waiter {
			select {
			case prev.waiter <- dsh.EventResultOutcome{Kind: "next"}:
			default:
			}
		}
		delete(s.eventsDeliveries, id)
	}
	if owner == nil {
		return nil
	}
	if s.eventsDeliveries == nil {
		s.eventsDeliveries = map[string]*eventsDelivery{}
	}
	ctx, cancel := context.WithCancel(owner.ctx)
	d := &eventsDelivery{owner: owner, waiter: waiter, ctx: ctx, cancel: cancel}
	s.eventsDeliveries[id] = d
	return d
}

func (s *Store) forgetEventsDeliveryLocked(id string) {
	d := s.eventsDeliveries[id]
	if d == nil {
		return
	}
	delete(s.eventsDeliveries, id)
	d.cancel()
	if d.responseCancel != nil {
		d.responseCancel()
	}
	if d.responseStop != nil {
		d.responseStop()
	}
}

func (s *Store) retireEventsDeliveryLocked(owner *EventsDeliveryOwner) {
	if owner == nil || owner.store != s || owner.retired {
		return
	}
	owner.retired = true
	owner.cancel()
	for id, d := range s.eventsDeliveries {
		if d.owner != owner {
			continue
		}
		waiter := d.waiter
		// dropPending remembers callId drafts and releases only this live claim.
		// Carrier retirement is not Host cancellation and creates no tombstone.
		if s.waiters[id] == waiter {
			delete(s.waiters, id)
			s.dropPending(id)
		} else {
			s.forgetEventsDeliveryLocked(id)
		}
		select {
		case waiter <- dsh.EventResultOutcome{Kind: "next"}:
		default:
		}
	}
	if s.eventsDeliveryOwner == owner {
		s.eventsDeliveryOwner = nil
	}
	s.releaseStalledAnswersLocked()
}

// releaseStalledAnswerLocked clears the sending flag of one card whose answer
// attempt is gone. A different attempt that holds the card keeps it.
func (s *Store) releaseStalledAnswerLocked(eventID string) {
	if _, inFlight := s.questionRecovery.inFlight[eventID]; inFlight {
		return
	}
	if p := s.findPending(eventID); p != nil && p.Sending {
		p.Sending, p.Error = false, QuestionDeliveryStopped
	}
}

// releaseStalledAnswersLocked clears the progress claim of any card that has no
// attempt and no live delivery left to finish it. A retired delivery generation
// is exactly that case: the channel is gone, so the card has to say so instead
// of looking like a slow send forever.
func (s *Store) releaseStalledAnswersLocked() {
	for i := range s.pending {
		p := &s.pending[i]
		// A continued question is answered through the question RPC attempt that
		// owns its Sending flag, not through this delivery generation.
		if !p.Sending || p.Continued {
			continue
		}
		if _, inFlight := s.questionRecovery.inFlight[p.EventID]; inFlight {
			continue
		}
		if d := s.eventsDeliveries[p.EventID]; d != nil && s.waiters[p.EventID] == d.waiter {
			continue
		}
		p.Sending, p.Error = false, QuestionDeliveryStopped
	}
}

func (s *Store) RetireEventsDelivery(owner *EventsDeliveryOwner) {
	s.mu.Lock()
	s.retireEventsDeliveryLocked(owner)
	s.mu.Unlock()
	s.persistQuestionDrafts()
	s.fire()
}

func expiredEventsOutcome(owner *EventsDeliveryOwner, id string) *dsh.EventResultOutcome {
	ctx := context.Background()
	var client *dsh.Client
	if owner != nil {
		ctx, client = owner.ctx, owner.client
	}
	return &dsh.EventResultOutcome{Kind: "next", Delivery: &dsh.EventDelivery{Client: client, EventID: id, Context: ctx, Valid: func() bool { return false }}}
}

func (s *Store) finishEventsOutcome(owner *EventsDeliveryOwner, id string, waiter chan dsh.EventResultOutcome, out dsh.EventResultOutcome) *dsh.EventResultOutcome {
	if owner == nil {
		return &out
	}
	s.mu.Lock()
	d := s.eventsDeliveries[id]
	if !s.eventsOwnerCurrentLocked(owner) || waiter != nil && (d == nil || d.owner != owner || d.waiter != waiter || s.waiters[id] != waiter) {
		s.mu.Unlock()
		return expiredEventsOutcome(owner, id)
	}
	if d == nil {
		// Automatic approval still owns an HTTP response, despite having no UI
		// waiter. Its acknowledgement must release this delivery's context too.
		s.registerEventsDeliveryLocked(owner, id, nil)
		d = s.eventsDeliveries[id]
	}
	base := owner.ctx
	scope := out.Delivery
	if d != nil {
		base = d.ctx
		if scope == d.response {
			scope = d.scope
		}
		if d.responseCancel != nil {
			d.responseCancel()
		}
		if d.responseStop != nil {
			d.responseStop()
		}
	}
	ctx, cancel := context.WithCancel(base)
	var stop func() bool
	if scope != nil {
		if scope.Context == nil {
			cancel()
		} else {
			stop = context.AfterFunc(scope.Context, cancel)
			if scope.Context.Err() != nil {
				cancel()
			}
		}
	}
	var authority *dsh.EventDelivery
	authority = &dsh.EventDelivery{Client: owner.client, EventID: id, Context: ctx, Valid: func() bool {
		if scope != nil && (scope.Client != owner.client || scope.EventID != id || scope.Context == nil || scope.Context.Err() != nil || scope.Valid == nil || !scope.Valid()) {
			return false
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.eventsOwnerCurrentLocked(owner) {
			return false
		}
		return waiter == nil || s.eventsDeliveries[id] == d && s.waiters[id] == waiter && d.response == authority
	}}
	out.Delivery = authority
	if d != nil {
		d.scope, d.response, d.responseCancel, d.responseStop = scope, authority, cancel, stop
		s.responses[id] = out
	} else {
		// No delivery owns the returned response context in this path. Release
		// both its cancellation registration and parent reference immediately.
		cancel()
		if stop != nil {
			stop()
		}
	}
	s.mu.Unlock()
	return &out
}

func (s *Store) eventsResultCurrentLocked(owner *EventsDeliveryOwner, result dsh.EventResult) bool {
	d := s.eventsDeliveries[result.EventID]
	return s.eventsOwnerCurrentLocked(owner) && d != nil && d.owner == owner && s.waiters[result.EventID] == d.waiter && result.Outcome.Delivery != nil && d.response == result.Outcome.Delivery && s.responses[result.EventID].Delivery == d.response
}

func (s *Store) HandleWaterfallForDelivery(owner *EventsDeliveryOwner, wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	if owner == nil {
		return expiredEventsOutcome(nil, wf.EventID), nil
	}
	return s.handleWaterfallForDelivery(owner, wf)
}
func (s *Store) OnResultForDelivery(owner *EventsDeliveryOwner, result dsh.EventResult, err error) bool {
	if owner == nil {
		return false
	}
	return s.onResultForDelivery(owner, result, err)
}
func (s *Store) OnCancelForDelivery(owner *EventsDeliveryOwner, event dsh.EventsCancel) {
	if owner != nil {
		s.onCancelForDelivery(owner, event)
	}
}

package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// PlanResourceTarget is the existing SessionAddress subagent variant plus the
// exact immutable invocation. Child IDs must never enter the ordinary route.
type PlanResourceTarget struct{ ParentSessionID, ChildSessionID, Mode, CallID string }

func (t PlanResourceTarget) valid() bool {
	return t.ParentSessionID != "" && t.ChildSessionID != "" && t.CallID != "" && (t.Mode == "one-shot" || t.Mode == "continuable" || t.Mode == "unknown")
}
func (t PlanResourceTarget) address() json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"kind": "subagent", "parentSessionId": t.ParentSessionID, "childSessionId": t.ChildSessionID, "mode": t.Mode})
	return raw
}

type PlanResourceReadState struct {
	Loading bool
	Plan    SubmittedPlan
	Error   string
}

// PlanResourceRead belongs to one Host binding and complete addressed target.
// It never enters Store.messages, changes selectedID, or persists document text.
type PlanResourceRead struct {
	mu      sync.Mutex
	binding string
	target  PlanResourceTarget
	state   PlanResourceReadState
	cancel  context.CancelFunc
	closed  bool
	owned   func() bool
	done    chan struct{}
}

func (r *PlanResourceRead) Matches(binding string, target PlanResourceTarget) bool {
	return r != nil && r.binding == binding && r.target == target
}
func (r *PlanResourceRead) Snapshot() PlanResourceReadState {
	if r == nil {
		return PlanResourceReadState{}
	}
	if r.owned != nil && !r.owned() {
		r.Cancel()
		return PlanResourceReadState{Error: "The plan read belongs to an obsolete Host/session binding"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}
func (r *PlanResourceRead) Cancel() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.closed = true
	r.state = PlanResourceReadState{}
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// ReadSubagentPlan reuses the real session/follow and session/page faces. Like
// the original resource provider, follow stops at its opening snapshot and all
// pages use that snapshot's fixed cursor and the SAME addressed Session.
// Addressed subagent follow only observes history and never sends a prompt.
// The Host promotes prepared Sessions only for ordinary kind:"session" addresses.
func (s *Store) ReadSubagentPlan(ctx context.Context, binding string, target PlanResourceTarget, notify func()) *PlanResourceRead {
	readCtx, cancel := context.WithCancel(ctx)
	r := &PlanResourceRead{binding: binding, target: target, cancel: cancel, done: make(chan struct{})}
	reject := func(message string) *PlanResourceRead { r.state.Error = message; cancel(); close(r.done); return r }
	if !target.valid() {
		return reject("Unsupported subagent plan resource address")
	}
	if s == nil {
		return reject("Session history is unavailable")
	}
	s.mu.Lock()
	client, mux, generation, selected, historyBinding := s.client, s.mux, s.followGeneration, s.selectedID, s.planHistoryBinding
	s.mu.Unlock()
	if client == nil || mux == nil || binding == "" || binding != fmt.Sprintf("%p\x00%s", client, selected) || selected != target.ParentSessionID || historyBinding != binding {
		return reject("Waiting for the current Host/parent session history; reopen the plan once connected")
	}
	r.owned = func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.client == client && s.mux == mux && s.selectedID == selected && s.followGeneration == generation && s.planHistoryBinding == binding
	}
	r.state.Loading = true
	go func() {
		defer close(r.done)
		defer cancel()
		plan, err := readAddressedPlan(readCtx, client, mux, target, r.owned)
		if !r.owned() {
			err = errors.New("The plan read belongs to an obsolete Host/session binding")
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			return
		}
		if readCtx.Err() != nil {
			err = readCtx.Err()
		}
		r.state.Loading = false
		if err != nil {
			r.state.Error = err.Error()
		} else {
			r.state.Plan = plan
		}
		r.mu.Unlock()
		if notify != nil {
			notify()
		}
	}()
	return r
}

func readAddressedPlan(ctx context.Context, client *dsh.Client, mux *dsh.Mux, target PlanResourceTarget, owned func() bool) (SubmittedPlan, error) {
	check := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !owned() {
			return errors.New("Obsolete plan resource binding")
		}
		return nil
	}
	if err := check(); err != nil {
		return SubmittedPlan{}, err
	}
	if err := mux.Dial(ctx); err != nil {
		return SubmittedPlan{}, err
	}
	if err := check(); err != nil {
		return SubmittedPlan{}, err
	}
	address := target.address()
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": api.SessionFollowRequest{Address: address}}})
	if err != nil {
		return SubmittedPlan{}, err
	}
	// Cancel explicitly at the snapshot, including malformed openings. No live
	// child frame can be folded into the selected parent's transcript.
	var snapshot struct {
		Type   string `json:"type"`
		Header struct {
			ID string `json:"id"`
		} `json:"header"`
		Cursor  *float64                     `json:"cursor"`
		Records []api.SessionPageRecordsItem `json:"records"`
		HasMore bool                         `json:"hasMore"`
	}
	for {
		raw, recvErr := stream.Recv(ctx)
		if recvErr != nil {
			stream.Cancel()
			return SubmittedPlan{}, fmt.Errorf("Plan opening snapshot: %w", recvErr)
		}
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			stream.Cancel()
			return SubmittedPlan{}, errors.New("Invalid plan history snapshot")
		}
		if snapshot.Type == "snapshot" {
			break
		}
	}
	stream.Cancel()
	if snapshot.Header.ID != target.ChildSessionID || snapshot.Cursor == nil || *snapshot.Cursor < 0 || snapshot.Records == nil {
		return SubmittedPlan{}, errors.New("Invalid addressed plan history snapshot")
	}
	cursor := *snapshot.Cursor
	page := api.SessionPage{Records: snapshot.Records, HasMore: snapshot.HasMore}
	beforeLimit := cursor + 1
	for {
		if err := check(); err != nil {
			return SubmittedPlan{}, err
		}
		for _, rec := range page.Records {
			if rec.Type != "event" || rec.Event.Seq <= 0 || rec.Event.Seq > cursor {
				continue
			}
			raw, _ := json.Marshal(rec.Event)
			if plan, ok := SubmittedPlanFromEvent(raw); ok && plan.CallID == target.CallID {
				return plan, nil
			}
		}
		if !page.HasMore || len(page.Records) == 0 {
			return SubmittedPlan{}, errors.New("The submitted plan was not found in this subagent Session")
		}
		before := page.Records[0].Event.Seq
		if before <= 0 || before >= beforeLimit {
			return SubmittedPlan{}, errors.New("Subagent plan history made no backward progress")
		}
		beforeLimit = before
		req := api.SessionPageRequest{Address: address, ThroughSeq: cursor, BeforeSeq: &before}
		var next api.SessionPage
		if err := client.Call(ctx, "session/page", map[string]any{"request": req}, &next); err != nil {
			return SubmittedPlan{}, err
		}
		page = next
	}
}

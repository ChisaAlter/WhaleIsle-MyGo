package proactive

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func (s *Service) Create(in CreateInput) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Autonomy == 0 {
		in.Autonomy = 1
	}
	if in.Autonomy < 1 || in.Autonomy > 5 {
		return Task{}, fmt.Errorf("autonomy must be an integer from 1 to 5")
	}
	if strings.TrimSpace(in.Title) == "" {
		return Task{}, fmt.Errorf("task title is required")
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	if id := s.createKeys[in.IdempotencyKey]; in.IdempotencyKey != "" && id != "" {
		if in.ID != "" && in.ID != id {
			return Task{}, fmt.Errorf("%w: idempotency key belongs to another task", ErrConflict)
		}
		in.ID = id
	}
	if in.ID == "" {
		in.ID = rand.Text()
	}
	if prior, ok := s.tasks[in.ID]; ok {
		if prior.Title == in.Title && prior.Description == in.Description && prior.Autonomy == in.Autonomy && prior.Source == in.Source && prior.SourceURL == in.SourceURL && prior.Kind == in.Kind && reflect.DeepEqual(prior.Labels, in.Labels) && prior.IdempotencyKey == in.IdempotencyKey {
			return clone(prior), nil
		}
		return Task{}, fmt.Errorf("%w: task identity already exists with different content", ErrConflict)
	}
	at := timestamp(s.now())
	t := Task{ID: in.ID, Title: in.Title, Description: in.Description, Autonomy: in.Autonomy, State: "discovered", Source: in.Source, SourceURL: in.SourceURL, Kind: in.Kind, Labels: in.Labels, IdempotencyKey: in.IdempotencyKey, CreatedAt: at, UpdatedAt: at}
	item, _ := json.Marshal(map[string]any{"id": t.ID, "taskId": t.ID, "title": t.Title, "source": t.Source, "autonomy": t.Autonomy, "state": t.State, "createdAt": t.CreatedAt})
	if err := s.commit(Event{Type: "task.created", Task: &t}, Event{Type: "inbox.upserted", Item: item}); err != nil {
		return Task{}, err
	}
	return clone(t), nil
}

func (s *Service) Transition(in TransitionInput) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validState(in.State) {
		return Task{}, fmt.Errorf("unknown task state %q", in.State)
	}
	t, ok := s.tasks[in.TaskID]
	if !ok {
		return Task{}, ErrNotFound
	}
	if in.IdempotencyKey != "" {
		if prior, exists := s.transitionKeys[in.TaskID+"\x00"+in.IdempotencyKey]; exists {
			if prior.State != in.State || prior.Reason != in.Reason {
				return Task{}, fmt.Errorf("%w: transition key reused for another action", ErrConflict)
			}
			return clone(t), nil
		}
	}
	if t.Terminal() {
		return Task{}, fmt.Errorf("%w: task is already %s", ErrConflict, t.State)
	}
	if t.Execution != nil && t.Execution.Unsettled() {
		return Task{}, fmt.Errorf("%w: stop or inspect the native execution before changing task state", ErrConflict)
	}
	for _, a := range s.approvals {
		if a.TaskID == t.ID && a.Delivery != nil && a.Delivery.Status == "sending" {
			return Task{}, fmt.Errorf("%w: a delivery is in flight", ErrConflict)
		}
	}
	if in.State == "waiting_approval" {
		return Task{}, fmt.Errorf("request an approval before entering waiting_approval")
	}
	if t.ApprovalID != "" && in.State != "cancelled" && in.State != "failed" {
		return Task{}, fmt.Errorf("%w: pending approval must be decided by the user", ErrConflict)
	}
	if in.Actor == "" {
		in.Actor = "system"
	}
	events := []Event{{Type: "task.transitioned", TaskID: t.ID, State: in.State, Actor: in.Actor, Reason: in.Reason, Evidence: in.Evidence, IdempotencyKey: in.IdempotencyKey}}
	terminal := Task{State: in.State}.Terminal()
	if terminal {
		events = append(events, s.cancelDependents(t.ID)...)
	}
	if err := s.commit(events...); err != nil {
		return Task{}, err
	}
	return clone(s.tasks[t.ID]), nil
}

func (s *Service) cancelDependents(taskID string) []Event {
	var events []Event
	for _, id := range s.approvalOrder {
		a := s.approvals[id]
		if a.TaskID == taskID && a.Delivery != nil && (a.Delivery.Status == "pending" || a.Delivery.Status == "ready") {
			d := clone(*a.Delivery)
			d.Status = "cancelled"
			events = append(events, Event{Type: "delivery.updated", ApprovalID: id, Delivery: &d})
		}
		if a.TaskID == taskID && a.Status == "pending" {
			events = append(events, Event{Type: "outbox.updated", ApprovalID: a.ID, Status: "cancelled", DecidedAt: timestamp(s.now()), DecidedBy: "task"})
		}
	}
	for _, id := range s.jobOrder {
		j := s.jobs[id]
		if j.TaskID == taskID && j.Status == "scheduled" {
			events = append(events, Event{Type: "schedule.cancelled", JobID: j.ID})
		}
	}
	return events
}

func sameJSON(a, b json.RawMessage) bool {
	if len(a) == 0 {
		a = json.RawMessage("null")
	}
	if len(b) == 0 {
		b = json.RawMessage("null")
	}
	var va, vb any
	da, db := json.NewDecoder(bytes.NewReader(a)), json.NewDecoder(bytes.NewReader(b))
	da.UseNumber()
	db.UseNumber()
	if da.Decode(&va) != nil || db.Decode(&vb) != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

func (s *Service) RequestApproval(in ApprovalInput) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[in.TaskID]
	if !ok {
		return Approval{}, ErrNotFound
	}
	if t.Terminal() {
		return Approval{}, fmt.Errorf("%w: cannot approve a terminal task", ErrConflict)
	}
	if strings.TrimSpace(in.Action) == "" {
		return Approval{}, fmt.Errorf("approval action is required")
	}
	if len(in.Target) != 0 && !json.Valid(in.Target) {
		return Approval{}, fmt.Errorf("approval target is invalid JSON")
	}
	if in.DataClass == "" {
		in.DataClass = "non-sensitive"
	}
	if in.Risk == "" {
		in.Risk = "low"
	}
	if t.ApprovalID != "" {
		prior := s.approvals[t.ApprovalID]
		if (in.ID == "" || in.ID == prior.ID) && prior.Action == in.Action && sameJSON(prior.Target, in.Target) && prior.DataClass == in.DataClass && prior.Risk == in.Risk {
			return clone(prior), nil
		}
		return Approval{}, fmt.Errorf("%w: another approval is pending for this task", ErrConflict)
	}
	if in.ID == "" {
		in.ID = rand.Text()
	}
	if _, exists := s.approvals[in.ID]; exists {
		return Approval{}, fmt.Errorf("%w: approval identity already exists", ErrConflict)
	}
	a := Approval{ID: in.ID, TaskID: t.ID, Action: in.Action, Target: bytes.Clone(in.Target), DataClass: in.DataClass, Risk: in.Risk, Status: "pending", CreatedAt: timestamp(s.now())}
	item, _ := json.Marshal(a)
	if err := s.commit(Event{Type: "approval.requested", Approval: &a}, Event{Type: "task.transitioned", TaskID: t.ID, State: "waiting_approval", Actor: "policy", Reason: "Human approval required"}, Event{Type: "outbox.upserted", Item: item}); err != nil {
		return Approval{}, err
	}
	return clone(a), nil
}

// Decide is deliberately absent from Dispatch. Only an explicit native user
// action may call it; the Host's install-control token is not human consent.
func (s *Service) Decide(approvalID, decision, note string) (Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if decision != "approved" && decision != "rejected" {
		return Approval{}, fmt.Errorf("decision must be approved or rejected")
	}
	a, ok := s.approvals[approvalID]
	if !ok {
		return Approval{}, ErrNotFound
	}
	if a.Status != "pending" {
		if a.Status == decision {
			return clone(a), nil
		}
		return Approval{}, fmt.Errorf("%w: approval is already %s", ErrConflict, a.Status)
	}
	t, ok := s.tasks[a.TaskID]
	if !ok || t.Terminal() && !(a.Delivery != nil && t.State == "delivered") || t.ApprovalID != a.ID {
		return Approval{}, fmt.Errorf("%w: approval no longer belongs to the active task", ErrConflict)
	}
	state, reason := "resumed", "Approval granted"
	if decision == "rejected" {
		state, reason = "failed", "Approval rejected"
	}
	if a.ToolCall != nil {
		state = "running"
		if decision == "rejected" {
			reason = "Tool operation rejected"
		}
	}
	if a.Delivery != nil {
		state = t.State
	}
	events := []Event{
		{Type: "approval.decided", ApprovalID: a.ID, Decision: decision, Actor: "user", Note: note},
		{Type: "outbox.updated", ApprovalID: a.ID, Status: decision, DecidedAt: timestamp(s.now()), DecidedBy: "user"},
		{Type: "task.transitioned", TaskID: a.TaskID, State: state, Actor: "user", Reason: reason},
	}
	if a.Delivery != nil {
		d := clone(*a.Delivery)
		d.Status = "ready"
		if decision == "rejected" {
			d.Status = "rejected"
		}
		events = append(events, Event{Type: "delivery.updated", ApprovalID: a.ID, Delivery: &d})
	}
	if decision == "rejected" && a.ToolCall == nil && a.Delivery == nil {
		for _, e := range s.cancelDependents(t.ID) {
			if e.ApprovalID != a.ID {
				events = append(events, e)
			}
		}
	}
	if err := s.commit(events...); err != nil {
		return Approval{}, err
	}
	return clone(s.approvals[a.ID]), nil
}

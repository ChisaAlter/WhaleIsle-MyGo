package session

import (
	"context"
	"encoding/json"
)

type PlanModeControl struct {
	Present, Active, Pending, Effective, Leaving bool
	Error                                        string
}
type planModeState struct {
	PlanModeControl
	sessionID  string
	generation uint64
	seq        float64
}

func (s *Store) planModeControlLocked() PlanModeControl {
	p := s.planModeControl
	if p.sessionID != s.selectedID || p.generation != s.followGeneration || s.selectedID == "" {
		return PlanModeControl{}
	}
	return p.PlanModeControl
}
func (s *Store) applyPlanModeValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	s.applyPlanModeSelectionLocked(all["plan"], seq)
}
func (s *Store) applyPlanModeSelectionLocked(raw json.RawMessage, seq float64) {
	p := &s.planModeControl
	if p.sessionID != s.selectedID || p.generation != s.followGeneration {
		*p = planModeState{sessionID: s.selectedID, generation: s.followGeneration}
	}
	if seq < p.seq {
		return
	}
	if len(raw) == 0 || string(raw) == "null" {
		p.PlanModeControl = PlanModeControl{}
		p.seq = seq
		return
	}
	var value struct {
		Active  *bool `json:"active"`
		Pending *bool `json:"pending"`
	}
	if json.Unmarshal(raw, &value) != nil || value.Active == nil || value.Pending == nil {
		return
	}
	p.seq = seq
	p.Present = true
	p.Active = *value.Active
	p.Pending = *value.Pending
	p.Effective = p.Active != p.Pending
}
func (s *Store) applyPlanModeControlLocked(raw json.RawMessage) {
	var frame struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Type != "projection" || frame.Key != "plan" || frame.SessionID != s.selectedID {
		return
	}
	s.applyPlanModeSelectionLocked(frame.Value, frame.Seq)
}
func (s *Store) ExitPlanMode(ctx context.Context, targetID string) {
	s.mu.Lock()
	state := s.planModeControlLocked()
	client, id, generation := s.client, s.selectedID, s.followGeneration
	if client == nil || id == "" || id != targetID || !state.Present || !state.Effective || state.Leaving || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.planModeControl.Leaving = true
	s.planModeControl.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		err := executeCommand(ctx, client, id, "/plan off", nil)
		s.mu.Lock()
		if !s.sessionResultCurrentLocked(ctx, client, id, generation) {
			s.mu.Unlock()
			return
		}
		s.planModeControl.Leaving = false
		if err != nil {
			s.planModeControl.Error = err.Error()
		}
		// A command acknowledgment is not a plan projection; the control watch owns
		// active/pending, including an effective off target before durable mode flips.
		s.mu.Unlock()
		s.fire()
	}()
}

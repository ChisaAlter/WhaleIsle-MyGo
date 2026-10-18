package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/workflowrun"
)

type WorkflowMemberView struct {
	Messages                        []Message
	PresentationReady               bool
	Token                           uint64
	Open, Busy                      bool
	ParentID, RunID, ChildID, Label string
	MemberSeq                       float64
	Process                         projectflow.ProcessSnapshot
}
type workflowMemberState struct {
	notifyPending bool
	view          WorkflowMemberView
	cancel        context.CancelFunc
	client        *dsh.Client
	mux           *dsh.Mux
	generation    uint64
	address       map[string]any
}

func (s *Store) invalidateWorkflowLocked() {
	s.workflowBoundaries = nil
	if s.workflowMember.cancel != nil {
		s.workflowMember.cancel()
	}
	token := s.workflowMember.view.Token + 1
	s.workflowMember = workflowMemberState{view: WorkflowMemberView{Token: token}}
}
func (s *Store) WorkflowMemberSnapshot() WorkflowMemberView {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflowMember.notifyPending = false
	v := s.workflowMember.view
	v.Messages = append([]Message(nil), v.Messages...)
	v.Process.Records = append([]projectflow.ProcessRecord(nil), v.Process.Records...)
	return v
}
func (s *Store) WorkflowMemberCanReconnect(token uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.workflowMember.view
	return v.Open && v.Token == token && s.planHistoryBinding != "" && s.workflowMemberAllowedLocked(v.ParentID, v.RunID, v.MemberSeq, v.ChildID)
}
func (s *Store) CloseWorkflowMember(token uint64) {
	s.mu.Lock()
	if s.workflowMember.view.Token != token {
		s.mu.Unlock()
		return
	}
	if s.workflowMember.cancel != nil {
		s.workflowMember.cancel()
	}
	s.workflowMember = workflowMemberState{view: WorkflowMemberView{Token: token + 1}}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) workflowMemberCurrentLocked(ctx context.Context, token uint64) bool {
	w := &s.workflowMember
	return ctx.Err() == nil && w.view.Open && w.view.Token == token && w.client == s.client && w.mux == s.mux && w.generation == s.followGeneration && w.view.ParentID == s.selectedID
}

// OpenWorkflowMember validates the original parent catalog before following a
// one-shot address. It never selects a bare child, activates it, or sends input.
func (s *Store) OpenWorkflowMember(binding, parent, runID string, memberSeq float64, child, label string) {
	s.mu.Lock()
	if binding == "" || binding != s.planHistoryBinding || s.client == nil || s.mux == nil || !s.workflowMemberAllowedLocked(parent, runID, memberSeq, child) {
		s.mu.Unlock()
		return
	}
	if s.workflowMember.cancel != nil {
		s.workflowMember.cancel()
	}
	var previous projectflow.ProcessSnapshot
	old := &s.workflowMember
	if old.view.ParentID == parent && old.view.RunID == runID && old.view.ChildID == child && old.client == s.client && old.mux == s.mux && old.generation == s.followGeneration {
		previous = old.view.Process
		previous.Live = false
	}
	ctx, cancel := context.WithCancel(context.Background())
	token := s.workflowMember.view.Token + 1
	client, mux := s.client, s.mux
	s.workflowMember = workflowMemberState{view: WorkflowMemberView{Token: token, Open: true, Busy: true, ParentID: parent, RunID: runID, MemberSeq: memberSeq, ChildID: child, Label: label, Process: previous}, cancel: cancel, client: client, mux: mux, generation: s.followGeneration}
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		query, cancelQuery := context.WithTimeout(ctx, 15*time.Second)
		var projection api.SessionProjectionsValue
		err := client.Call(query, "session/projections", map[string]any{"request": map[string]any{"sessionId": parent}}, &projection)
		cancelQuery()
		address, valid := workflowrun.OneShotAddress(parent, child, projection.Values)
		if err == nil && !valid {
			err = fmt.Errorf("original one-shot member is absent from the parent catalog")
		}
		if err != nil {
			s.finishWorkflowMember(ctx, token, err)
			return
		}
		s.mu.Lock()
		if !s.workflowMemberCurrentLocked(ctx, token) || !s.workflowMemberAllowedLocked(parent, runID, memberSeq, child) {
			s.mu.Unlock()
			s.finishWorkflowMember(ctx, token, fmt.Errorf("the member is no longer running"))
			return
		}
		s.workflowMember.address = address
		s.mu.Unlock()
		s.followWorkflowMember(ctx, token, mux, address)
	}()
}
func (s *Store) finishWorkflowMember(ctx context.Context, token uint64, err error) {
	s.mu.Lock()
	if !s.workflowMemberCurrentLocked(ctx, token) {
		s.mu.Unlock()
		return
	}
	w := &s.workflowMember
	w.view.Busy = false
	w.view.Process.Live = false
	if err != nil {
		w.view.Process.Error = err.Error()
	}
	w.cancel = nil
	s.mu.Unlock()
	s.fire()
}

// EarlierWorkflowMember freezes the current observer before reading an older
// page at its captured inclusive log cut. Back-to-latest is an explicit reopen.
func (s *Store) EarlierWorkflowMember(token uint64) {
	s.mu.Lock()
	w := &s.workflowMember
	if w.view.Token != token || !w.view.Open || w.view.Busy || w.address == nil || !w.view.Process.HasMore {
		s.mu.Unlock()
		return
	}
	before := float64(0)
	for _, r := range w.view.Process.Records {
		if r.Type == "event" && r.Event.Seq > 0 && (before == 0 || r.Event.Seq < before) {
			before = r.Event.Seq
		}
	}
	through := w.view.Process.Cursor
	if before <= 0 || through <= 0 {
		s.mu.Unlock()
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.view.Token++
	token = w.view.Token
	w.view.Busy = true
	w.view.Process.Live = false
	w.view.Messages = nil
	w.view.PresentationReady = false
	client, address := w.client, w.address
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		query, cancelQuery := context.WithTimeout(ctx, 15*time.Second)
		defer cancelQuery()
		var page projectflow.ProcessSnapshot
		err := client.Call(query, "session/page", map[string]any{"request": map[string]any{"address": address, "throughSeq": through, "beforeSeq": before, "maxMessages": 30}}, &page)
		if err == nil && page.Records == nil {
			err = fmt.Errorf("invalid member history page")
		}
		s.mu.Lock()
		if !s.workflowMemberCurrentLocked(ctx, token) {
			s.mu.Unlock()
			return
		}
		w := &s.workflowMember
		w.view.Busy = false
		w.cancel = nil
		if err != nil {
			w.view.Process.Error = err.Error()
		} else {
			if len(page.Records) > projectflow.ProcessRecordLimit {
				page.Records = append([]projectflow.ProcessRecord(nil), page.Records[len(page.Records)-projectflow.ProcessRecordLimit:]...)
				page.Truncated = true
				page.HasMore = true
			}
			page.Type = "snapshot"
			page.Cursor = through
			if len(page.Records) == 0 {
				page.HasMore = false
			}
			w.view.Process = page
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// WorkflowProcessText is a copy/export representation of visible durable data.
func WorkflowProcessText(v WorkflowMemberView) string {
	var out strings.Builder
	for _, r := range v.Process.Records {
		if r.Type != "event" {
			continue
		}
		var payload any
		body := string(r.Event.Data)
		if json.Unmarshal(r.Event.Data, &payload) == nil {
			pretty, _ := json.MarshalIndent(payload, "", "  ")
			body = string(pretty)
		}
		fmt.Fprintf(&out, "#%.0f %s\n%s\n\n", r.Event.Seq, r.Event.Type, body)
	}
	return out.String()
}

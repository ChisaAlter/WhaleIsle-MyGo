// Package interaction holds pending Host waterfall requests (approval, questions).
package interaction

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

const (
	EventApproval        = "approval/request"
	EventUserQuestions   = "user-questions/request"
	outcomeAllowedOnce   = "allowed-once"
	outcomeAllowedAlways = "allowed-always"
	outcomeRejected      = "rejected"
)

// Pending is one waterfall waiting for a UI or policy answer.
type Pending struct {
	EventID string
	Event   string
	AgentID string
	Title   string
	Detail  string
	// Args is a monospace preview of tool input (approval cards).
	Args string
	// CallID is the harness tool call id when present.
	CallID string
}

// ViewState is UI-safe interaction state.
type ViewState struct {
	AutoAllow     bool
	Banner        string
	Pending       []Pending
	QuestionCards map[string]QuestionCardView
}

// Store serializes waterfall handlers with the main-thread UI.
type Store struct {
	mu             sync.Mutex
	notify         func()
	client         *dsh.Client
	mux            *dsh.Mux
	autoAllow      bool
	banner         string
	pending        []Pending
	waiters        map[string]chan dsh.EventResultOutcome
	questionDrafts map[string]*draftSession
}

// New returns an interaction store with manual tool approval (Electron default).
func New() *Store {
	return &Store{
		autoAllow:      false,
		banner:         "",
		waiters:        make(map[string]chan dsh.EventResultOutcome),
		questionDrafts: make(map[string]*draftSession),
	}
}

// Attach sets the UI refresh hook.
func (s *Store) Attach(notify func()) {
	s.mu.Lock()
	s.notify = notify
	s.mu.Unlock()
}

// Detach clears pending state when harness stops.
func (s *Store) Detach() {
	s.mu.Lock()
	s.client = nil
	s.mux = nil
	for id, ch := range s.waiters {
		select {
		case ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "client stopped"}}:
		default:
		}
		delete(s.waiters, id)
	}
	s.pending = nil
	s.banner = ""
	s.questionDrafts = make(map[string]*draftSession)
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		AutoAllow:     s.autoAllow,
		Banner:        s.banner,
		QuestionCards: make(map[string]QuestionCardView),
	}
	st.Pending = append([]Pending(nil), s.pending...)
	for id, ds := range s.questionDrafts {
		st.QuestionCards[id] = ds.cardView(id)
	}
	return st
}

// SetAutoAllow toggles immediate allow for tool approvals.
func (s *Store) SetAutoAllow(on bool) {
	s.mu.Lock()
	s.autoAllow = on
	if on {
		s.banner = ""
	}
	s.mu.Unlock()
	s.fire()
}

// ResolveAllow answers the oldest or matching pending approval with allowed-once.
func (s *Store) ResolveAllow(eventID string) {
	s.resolve(eventID, approvalOutcome(outcomeAllowedOnce))
}

// ResolveAllowAlways answers with allowed-always and enables local auto-allow.
func (s *Store) ResolveAllowAlways(eventID string) {
	s.mu.Lock()
	s.autoAllow = true
	s.banner = ""
	s.mu.Unlock()
	s.resolve(eventID, approvalOutcome(outcomeAllowedAlways))
}

// ResolveDeny rejects a pending approval or question.
func (s *Store) ResolveDeny(eventID string) {
	s.resolve(eventID, approvalOutcome(outcomeRejected))
}

// ResolveQuestions submits the user's option selections for a question waterfall.
func (s *Store) ResolveQuestions(eventID string) {
	s.mu.Lock()
	ds := s.questionDrafts[eventID]
	s.mu.Unlock()
	if ds == nil {
		s.mu.Lock()
		p := s.findPending(eventID)
		s.mu.Unlock()
		if p == nil {
			return
		}
		var req userQuestionsRequest
		_ = json.Unmarshal([]byte(p.Detail), &req)
		ds = newDraftSession(req)
	}
	if !ds.answerValid() {
		return
	}
	answer := ds.buildAnswer()
	raw, err := json.Marshal(answer)
	if err != nil {
		s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: err.Error()}})
		return
	}
	s.mu.Lock()
	p := s.findPending(eventID)
	agentID, callID := "", ""
	if p != nil {
		agentID = p.AgentID
		callID = p.CallID
	}
	s.mu.Unlock()
	if callID != "" {
		var rpcAns api.AskUserQuestionAnswer
		_ = json.Unmarshal(raw, &rpcAns)
		s.AnswerUserQuestionsRPC(context.Background(), agentID, callID, rpcAns)
	}
	s.resolve(eventID, dsh.EventResultOutcome{Kind: "result", Value: raw})
}

// SetQuestionSingleSelect picks one option for a single-select question.
func (s *Store) SetQuestionSingleSelect(eventID, questionID, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	ad.selected = []string{label}
	ad.selectWire = label
	for l, ptr := range ad.checks {
		if ptr != nil {
			*ptr = l == label
		}
	}
}

// SetQuestionOptionChecked toggles one option (checkbox multi-select or radio-style single).
func (s *Store) SetQuestionOptionChecked(eventID, questionID, label string, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ds := s.questionDrafts[eventID]
	if ds == nil {
		return
	}
	var multi bool
	for _, q := range ds.req.Questions {
		if q.ID == questionID {
			multi = q.MultiSelect
			break
		}
	}
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	if multi {
		if ptr := ad.checks[label]; ptr != nil {
			*ptr = on
		}
		if on {
			ad.selected = appendUnique(ad.selected, label)
		} else {
			ad.selected = removeLabel(ad.selected, label)
		}
		return
	}
	if on {
		ad.selected = []string{label}
		ad.selectWire = label
		for l, ptr := range ad.checks {
			if ptr != nil {
				*ptr = l == label
			}
		}
	}
}

// SetQuestionCustom sets free-text when a question has no options.
func (s *Store) SetQuestionCustom(eventID, questionID, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		return
	}
	ad.custom = text
}

// QuestionSingleSelectPtr returns a stable select binding for one question.
func (s *Store) QuestionSingleSelectPtr(eventID, questionID string) *string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		empty := ""
		return &empty
	}
	return &ad.selectWire
}

// QuestionCustomPtr returns a stable text binding for a free-text question.
func (s *Store) QuestionCustomPtr(eventID, questionID string) *string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		empty := ""
		return &empty
	}
	return &ad.custom
}

// QuestionOptionCheckedPtr returns a stable checkbox binding for one option label.
func (s *Store) QuestionOptionCheckedPtr(eventID, questionID, label string) *bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ad := s.answerDraftLocked(eventID, questionID)
	if ad == nil {
		f := false
		return &f
	}
	if ad.checks == nil {
		ad.checks = make(map[string]*bool)
	}
	if ptr, ok := ad.checks[label]; ok && ptr != nil {
		return ptr
	}
	v := containsLabel(ad.selected, label)
	ad.checks[label] = &v
	return ad.checks[label]
}

// QuestionsCanSubmit reports whether all answers are filled for a pending card.
func (s *Store) QuestionsCanSubmit(eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ds := s.questionDrafts[eventID]
	if ds == nil {
		return false
	}
	return ds.answerValid()
}

func (s *Store) answerDraftLocked(eventID, questionID string) *answerDraft {
	ds := s.questionDrafts[eventID]
	if ds == nil {
		return nil
	}
	ad := ds.answers[questionID]
	if ad == nil {
		ad = &answerDraft{checks: make(map[string]*bool)}
		ds.answers[questionID] = ad
	}
	return ad
}

func (s *Store) resolve(eventID string, outcome dsh.EventResultOutcome) {
	s.mu.Lock()
	if eventID == "" && len(s.pending) > 0 {
		eventID = s.pending[0].EventID
	}
	ch, ok := s.waiters[eventID]
	if ok {
		delete(s.waiters, eventID)
	}
	s.dropPending(eventID)
	s.mu.Unlock()
	if ok {
		ch <- outcome
	}
	s.fire()
}

func (s *Store) dropPending(eventID string) {
	if eventID == "" {
		s.pending = nil
		s.questionDrafts = make(map[string]*draftSession)
		return
	}
	delete(s.questionDrafts, eventID)
	out := s.pending[:0]
	for _, p := range s.pending {
		if p.EventID != eventID {
			out = append(out, p)
		}
	}
	s.pending = out
}

func (s *Store) findPending(eventID string) *Pending {
	for i := range s.pending {
		if s.pending[i].EventID == eventID {
			return &s.pending[i]
		}
	}
	if len(s.pending) > 0 {
		return &s.pending[0]
	}
	return nil
}

// HandleWaterfall implements dsh.EventsHandler waterfall dispatch for the shell.
func (s *Store) HandleWaterfall(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	switch wf.Event {
	case EventApproval:
		return s.handleApproval(wf)
	case EventUserQuestions:
		return s.handleUserQuestions(wf)
	default:
		return nil, nil
	}
}

func (s *Store) handleApproval(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	var req approvalRequest
	_ = json.Unmarshal(wf.Request, &req)
	title := req.ToolName
	if title == "" {
		title = wf.Event
	}
	detail := req.Reason
	if req.DisplayReason != "" {
		detail = req.DisplayReason
	}
	args := approvalArgsPreview(req)
	callID := req.CallID
	if callID == "" {
		callID = req.RequestID
	}

	s.mu.Lock()
	auto := s.autoAllow
	s.mu.Unlock()
	if auto {
		s.mu.Lock()
		s.banner = title
		s.mu.Unlock()
		s.fire()
		out := approvalOutcome(outcomeAllowedOnce)
		return &out, nil
	}

	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	s.waiters[wf.EventID] = ch
	s.pending = append(s.pending, Pending{
		EventID: wf.EventID,
		Event:   wf.Event,
		AgentID: wf.AgentID,
		Title:   title,
		Detail:  detail,
		Args:    args,
		CallID:  callID,
	})
	s.mu.Unlock()
	s.fire()

	out := <-ch
	return &out, nil
}

func (s *Store) handleUserQuestions(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	var req userQuestionsRequest
	_ = json.Unmarshal(wf.Request, &req)
	title := questionCardTitle(req)
	detail := string(wf.Request)
	callID := strings.TrimSpace(req.CallID)

	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	mux := s.mux
	s.waiters[wf.EventID] = ch
	s.questionDrafts[wf.EventID] = newDraftSession(req)
	s.pending = append(s.pending, Pending{
		EventID: wf.EventID,
		Event:   wf.Event,
		AgentID: wf.AgentID,
		Title:   title,
		Detail:  detail,
		CallID:  callID,
	})
	s.mu.Unlock()
	s.fire()
	if callID != "" {
		s.AttachUserQuestionsWait(context.Background(), mux, wf.AgentID, callID)
	}

	out := <-ch
	return &out, nil
}

// OnCancel drops a pending waterfall when the host cancels it.
func (s *Store) OnCancel(c dsh.EventsCancel) {
	s.mu.Lock()
	ch, ok := s.waiters[c.EventID]
	if ok {
		delete(s.waiters, c.EventID)
	}
	s.dropPending(c.EventID)
	s.mu.Unlock()
	if ok {
		ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "cancelled"}}
	}
	s.fire()
}

func approvalOutcome(decision string) dsh.EventResultOutcome {
	raw, _ := json.Marshal(decision)
	return dsh.EventResultOutcome{Kind: "result", Value: raw}
}

func defaultQuestionAnswer(req userQuestionsRequest) userQuestionsAnswer {
	var answers []userQuestionsAnswerItem
	for _, q := range req.Questions {
		item := userQuestionsAnswerItem{ID: q.ID}
		if len(q.Options) > 0 {
			item.Selected = []string{q.Options[0].Label}
		} else {
			item.Custom = strPtr("Continue")
		}
		answers = append(answers, item)
	}
	return userQuestionsAnswer{Answers: answers}
}

func strPtr(s string) *string { return &s }

type approvalRequest struct {
	ToolName      string          `json:"toolName"`
	Reason        string          `json:"reason"`
	DisplayReason string          `json:"displayReason"`
	RequestID     string          `json:"requestId"`
	CallID        string          `json:"callId"`
	Input         json.RawMessage `json:"input"`
	Args          json.RawMessage `json:"args"`
	Parameters    json.RawMessage `json:"parameters"`
	Command       string          `json:"command"`
	Description   string          `json:"description"`
}

func approvalArgsPreview(req approvalRequest) string {
	for _, raw := range []json.RawMessage{req.Input, req.Args, req.Parameters} {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, raw, "", "  "); err == nil && pretty.Len() > 0 {
			s := pretty.String()
			if len(s) > 2400 {
				return s[:2400] + "\n…"
			}
			return s
		}
		s := string(raw)
		if len(s) > 2400 {
			return s[:2400] + "…"
		}
		return s
	}
	if req.Command != "" {
		return req.Command
	}
	if req.Description != "" {
		return req.Description
	}
	return ""
}

type userQuestionsRequest struct {
	CallID    string             `json:"callId"`
	Questions []userQuestionItem `json:"questions"`
}

type userQuestionItem struct {
	ID          string               `json:"id"`
	Question    string               `json:"question"`
	Header      string               `json:"header"`
	MultiSelect bool                 `json:"multiSelect"`
	Options     []userQuestionOption `json:"options"`
}

func questionCardTitle(req userQuestionsRequest) string {
	if len(req.Questions) == 0 {
		return "Question"
	}
	if len(req.Questions) == 1 {
		if req.Questions[0].Header != "" {
			return req.Questions[0].Header
		}
		if req.Questions[0].Question != "" {
			return req.Questions[0].Question
		}
		return "Question"
	}
	if req.Questions[0].Header != "" {
		return req.Questions[0].Header
	}
	return "Questions"
}

type userQuestionOption struct {
	Label string `json:"label"`
}

type userQuestionsAnswer struct {
	Answers []userQuestionsAnswerItem `json:"answers"`
}

type userQuestionsAnswerItem struct {
	ID       string   `json:"id"`
	Selected []string `json:"selected,omitempty"`
	Custom   *string  `json:"custom,omitempty"`
}

func (s *Store) fire() {
	s.mu.Lock()
	n := s.notify
	s.mu.Unlock()
	if n != nil {
		n()
	}
}

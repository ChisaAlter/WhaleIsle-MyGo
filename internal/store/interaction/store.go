// Package interaction holds pending Host waterfall requests (approval, questions).
package interaction

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
)

const (
	EventApproval      = "approval/request"
	EventUserQuestions = "user-questions/request"
	outcomeAllowedOnce = "allowed-once"
	outcomeRejected    = "rejected"
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
	CallID  string
	Sending bool
	Error   string
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
	responses      map[string]dsh.EventResultOutcome
	cancelled      map[string]bool
	questionDrafts map[string]*draftSession
	waitCancels    map[string]context.CancelFunc
}

// New returns an interaction store with manual tool approval (Electron default).
func New() *Store {
	return &Store{
		autoAllow:      false,
		banner:         "",
		waiters:        make(map[string]chan dsh.EventResultOutcome),
		responses:      make(map[string]dsh.EventResultOutcome),
		cancelled:      make(map[string]bool),
		questionDrafts: make(map[string]*draftSession),
		waitCancels:    make(map[string]context.CancelFunc),
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
	for id, cancel := range s.waitCancels {
		cancel()
		delete(s.waitCancels, id)
	}
	for id, ch := range s.waiters {
		s.cancelled[id] = true
		select {
		case ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "client stopped"}}:
		default:
		}
		delete(s.waiters, id)
	}
	s.pending = nil
	s.responses = make(map[string]dsh.EventResultOutcome)
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

// ResolveAllowAlways is retained for callers compiled against the old shell.
// The Host protocol grants only one action; it never enables automatic approval.
func (s *Store) ResolveAllowAlways(eventID string) { s.ResolveAllow(eventID) }

// ResolveDeny rejects a pending approval or question.
func (s *Store) ResolveDeny(eventID string) {
	s.mu.Lock()
	p := s.findPending(eventID)
	question := p != nil && p.Event == EventUserQuestions
	s.mu.Unlock()
	if question {
		s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "UserQuestionError", Code: "ASK_CANCELLED", Message: "User declined to answer"}})
		return
	}
	s.resolve(eventID, approvalOutcome(outcomeRejected))
}

// ResolveQuestions submits the user's option selections for a question waterfall.
func (s *Store) ResolveQuestions(eventID string) {
	s.mu.Lock()
	ds := s.questionDrafts[eventID]
	if ds == nil || !ds.answerValid() {
		s.mu.Unlock()
		return
	}
	answer := ds.buildAnswer()
	s.mu.Unlock()

	raw, err := json.Marshal(answer)
	if err != nil {
		s.resolve(eventID, dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: err.Error()}})
		return
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
	ad.custom = ""
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
		ad.custom = ""
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
	if strings.TrimSpace(text) != "" {
		ds := s.questionDrafts[eventID]
		for _, q := range ds.req.Questions {
			if q.ID == questionID && !q.MultiSelect {
				ad.selected = nil
				ad.selectWire = ""
				for _, ptr := range ad.checks {
					if ptr != nil {
						*ptr = false
					}
				}
			}
		}
	}
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
	p := s.findPending(eventID)
	if p == nil || p.Sending {
		s.mu.Unlock()
		return
	}
	eventID = p.EventID
	ch := s.waiters[eventID]
	if ch == nil {
		s.mu.Unlock()
		return
	}
	p.Sending, p.Error = true, ""
	s.responses[eventID] = outcome
	ch <- outcome
	s.mu.Unlock()
	s.fire()
}

// OnResult retains the card and its exact response on delivery failure.
// Returning true lets the event stream wait for an explicit user retry.
func (s *Store) OnResult(result dsh.EventResult, err error) bool {
	s.mu.Lock()
	p := s.findPending(result.EventID)
	retry := false
	if p != nil {
		if err == nil {
			delete(s.waiters, result.EventID)
			s.dropPending(result.EventID)
		} else {
			p.Sending = false
			p.Error = err.Error()
			retry = true
		}
	}
	s.mu.Unlock()
	s.fire()
	return retry
}

func (s *Store) Retry(eventID string) {
	s.mu.Lock()
	p := s.findPending(eventID)
	outcome, ok := s.responses[eventID]
	ok = ok && p != nil && p.Error != ""
	s.mu.Unlock()
	if ok {
		s.resolve(eventID, outcome)
	}
}

// SnapshotForAgent prevents a request from appearing in another conversation.
func (s *Store) SnapshotForAgent(agentID string) ViewState {
	snap := s.Snapshot()
	pending := snap.Pending[:0]
	cards := make(map[string]QuestionCardView)
	for _, p := range snap.Pending {
		if p.AgentID == agentID && agentID != "" {
			pending = append(pending, p)
			if card, ok := snap.QuestionCards[p.EventID]; ok {
				cards[p.EventID] = card
			}
		}
	}
	snap.Pending, snap.QuestionCards = pending, cards
	return snap
}

func (s *Store) dropPending(eventID string) {
	if eventID == "" {
		s.pending = nil
		s.questionDrafts = make(map[string]*draftSession)
		return
	}
	if cancel := s.waitCancels[eventID]; cancel != nil {
		cancel()
		delete(s.waitCancels, eventID)
	}
	delete(s.responses, eventID)
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
	if eventID == "" && len(s.pending) > 0 {
		return &s.pending[0]
	}
	return nil
}

// HandleWaterfall implements dsh.EventsHandler waterfall dispatch for the shell.
func (s *Store) HandleWaterfall(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	s.mu.Lock()
	ch := s.waiters[wf.EventID]
	cancelled := s.cancelled[wf.EventID]
	s.mu.Unlock()
	if cancelled {
		out := approvalOutcome("cancelled")
		return &out, nil
	}
	if ch != nil {
		out := <-ch
		return &out, nil
	}

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
	if len(req.DisplayReason) > 0 {
		var localized map[string]string
		var plain string
		if json.Unmarshal(req.DisplayReason, &plain) == nil && plain != "" {
			detail = plain
		} else if json.Unmarshal(req.DisplayReason, &localized) == nil && localized["en"] != "" {
			detail = localized["en"]
		}
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
	if s.cancelled[wf.EventID] {
		s.mu.Unlock()
		out := approvalOutcome("cancelled")
		return &out, nil
	}
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
	if callID == "" && req.Wait != nil {
		callID = req.Wait.CallID
	}

	ch := make(chan dsh.EventResultOutcome, 1)
	s.mu.Lock()
	mux := s.mux
	if s.cancelled[wf.EventID] {
		s.mu.Unlock()
		out := approvalOutcome("cancelled")
		return &out, nil
	}
	s.waiters[wf.EventID] = ch
	s.questionDrafts[wf.EventID] = newDraftSession(req)
	waitCtx, cancelWait := context.WithCancel(context.Background())
	s.waitCancels[wf.EventID] = cancelWait
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
	if callID != "" && req.Wait != nil && req.Wait.Timed {
		s.attachQuestionWait(waitCtx, mux, wf.AgentID, callID, wf.EventID)
	}

	out := <-ch
	return &out, nil
}

// OnCancel drops a pending waterfall when the host cancels it.
func (s *Store) OnCancel(c dsh.EventsCancel) {
	s.mu.Lock()
	s.cancelled[c.EventID] = true
	ch, ok := s.waiters[c.EventID]
	if ok {
		delete(s.waiters, c.EventID)
	}
	s.dropPending(c.EventID)
	s.mu.Unlock()
	if ok {
		select {
		case ch <- dsh.EventResultOutcome{Kind: "rejected", Error: &dsh.EventRejection{Name: "Error", Message: "cancelled"}}:
		default:
		}
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
	DisplayReason json.RawMessage `json:"displayReason"`
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
	CallID string `json:"callId"`
	Wait   *struct {
		CallID string `json:"callId"`
		Timed  bool   `json:"timed"`
	} `json:"wait"`
	Questions []userQuestionItem `json:"questions"`
}

type userQuestionItem struct {
	ID          string               `json:"id"`
	Question    string               `json:"question"`
	Detail      string               `json:"detail"`
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
	Label       string `json:"label"`
	Description string `json:"description"`
}

type userQuestionsAnswer struct {
	Answers []userQuestionsAnswerItem `json:"answers"`
}

type userQuestionsAnswerItem struct {
	ID       string   `json:"id"`
	Selected []string `json:"selected"`
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

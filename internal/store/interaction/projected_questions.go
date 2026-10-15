package interaction

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type projectedQuestion struct {
	CallID    string             `json:"callId"`
	State     string             `json:"state"`
	Questions []userQuestionItem `json:"questions"`
}
type submittedQuestion struct {
	queuedSeen bool
	asOfSeq    float64
}
type questionAgentProjection struct {
	active                     []projectedQuestion
	settled                    map[string]bool
	queued                     map[string]bool
	submitted                  map[string]submittedQuestion
	answered                   map[string]bool
	drafts                     map[string]*draftSession
	questionsReady, inboxReady bool
	questionsSeq, inboxSeq     float64
	status                     string
}
type questionRecovery struct {
	agents    map[string]*questionAgentProjection
	epoch     uint64
	nextToken uint64
	inFlight  map[string]uint64
}

func questionCardID(agentID, callID string) string {
	return "continued-question:" + strconv.Itoa(len(agentID)) + ":" + agentID + callID
}
func (s *Store) questionAgentLocked(agentID string) *questionAgentProjection {
	if s.questionRecovery.agents == nil {
		s.questionRecovery.agents = make(map[string]*questionAgentProjection)
	}
	p := s.questionRecovery.agents[agentID]
	if p == nil {
		p = &questionAgentProjection{settled: map[string]bool{}, queued: map[string]bool{}, submitted: map[string]submittedQuestion{}, answered: map[string]bool{}, drafts: map[string]*draftSession{}}
		s.questionRecovery.agents[agentID] = p
	}
	return p
}

// SyncProjectedQuestions runs before SnapshotForAgent on the UI thread. It does
// not fire another frame. Only the Host's continued projection creates RPC cards;
// a local timeout by itself never guesses that the Host accepted continuation.
func (s *Store) SyncProjectedQuestions(agentID string, questions, inbox json.RawMessage, sequences ...float64) {
	if strings.TrimSpace(agentID) == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.questionAgentLocked(agentID)
	hasSeq := len(sequences) == 2
	if len(questions) != 0 && (!hasSeq || !p.questionsReady || sequences[0] >= p.questionsSeq) {
		var face struct {
			Active  []projectedQuestion `json:"active"`
			Settled []struct {
				CallID string `json:"callId"`
			} `json:"settled"`
		}
		if json.Unmarshal(questions, &face) == nil && face.Active != nil {
			p.active, p.questionsReady = face.Active, true
			if hasSeq {
				p.questionsSeq = sequences[0]
			}
			p.settled = make(map[string]bool, len(face.Settled))
			for _, row := range face.Settled {
				p.settled[row.CallID] = true
			}
		}
	}
	if len(inbox) != 0 && (!hasSeq || !p.inboxReady || sequences[1] >= p.inboxSeq) {
		var lanes map[string]json.RawMessage
		if json.Unmarshal(inbox, &lanes) == nil {
			queued := make(map[string]bool)
			valid, found := true, false
			for _, key := range []string{"next-step", "next-turn"} {
				raw, ok := lanes[key]
				if !ok {
					continue
				}
				found = true
				var rows []struct {
					Source struct {
						Kind   string `json:"kind"`
						CallID string `json:"callId"`
					} `json:"source"`
				}
				if json.Unmarshal(raw, &rows) != nil {
					valid = false
					break
				}
				for _, row := range rows {
					if row.Source.Kind == "user-question-reply" && row.Source.CallID != "" {
						queued[row.Source.CallID] = true
					}
				}
			}
			if valid && found {
				p.queued, p.inboxReady = queued, true
				if hasSeq {
					p.inboxSeq = sequences[1]
				}
			}
		}
	}
	s.reconcileProjectedQuestionsLocked(agentID)
}

func (s *Store) reconcileProjectedQuestionsLocked(agentID string) {
	p := s.questionRecovery.agents[agentID]
	if p == nil || !p.questionsReady || !p.inboxReady {
		return
	}
	desired := make(map[string]projectedQuestion)
	active := make(map[string]bool)
	for _, row := range p.active {
		if row.CallID == "" {
			continue
		}
		active[row.CallID] = true
		if submission, ok := p.submitted[row.CallID]; ok {
			if p.queued[row.CallID] {
				submission.queuedSeen = true
				p.submitted[row.CallID] = submission
			} else if submission.queuedSeen || (p.questionsSeq > submission.asOfSeq && p.inboxSeq > submission.asOfSeq) {
				delete(p.submitted, row.CallID)
				p.status = ""
			} // A discarded queued reply becomes answerable again.
		}
		_, submitted := p.submitted[row.CallID]
		if row.State == "continued" && len(row.Questions) > 0 && !p.queued[row.CallID] && !p.settled[row.CallID] && !p.answered[row.CallID] && !submitted {
			desired[row.CallID] = row
		}
	}
	for callID := range p.submitted {
		if !active[callID] || p.settled[callID] {
			delete(p.submitted, callID)
		}
	}
	// Recovered rows may be withdrawn, but a live waterfall keeps its own lifecycle.
	for _, pending := range append([]Pending(nil), s.pending...) {
		if pending.AgentID == agentID && pending.Continued {
			if _, ok := desired[pending.CallID]; !ok {
				s.dropPending(pending.EventID)
			}
		}
	}
	for _, row := range p.active {
		if _, ok := desired[row.CallID]; !ok {
			continue
		}
		exists := false
		for _, pending := range s.pending {
			if pending.AgentID == agentID && pending.Event == EventUserQuestions && pending.CallID == row.CallID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		req := userQuestionsRequest{CallID: row.CallID, Questions: row.Questions}
		id := questionCardID(agentID, row.CallID)
		draft := p.drafts[row.CallID]
		if draft == nil {
			draft = newDraftSession(req)
			p.drafts[row.CallID] = draft
		}
		s.questionDrafts[id] = draft
		s.pending = append(s.pending, Pending{EventID: id, Event: EventUserQuestions, AgentID: agentID, CallID: row.CallID, Title: questionCardTitle(req), Continued: true})
	}
}

func (s *Store) rememberQuestionDraftLocked(eventID string) {
	pending := s.findPending(eventID)
	if pending == nil || pending.Event != EventUserQuestions || pending.CallID == "" {
		return
	}
	if draft := s.questionDrafts[eventID]; draft != nil {
		s.questionAgentLocked(pending.AgentID).drafts[pending.CallID] = draft
	}
}
func (s *Store) liveQuestionDraftLocked(agentID, callID string, req userQuestionsRequest) *draftSession {
	if callID == "" {
		return newDraftSession(req)
	}
	p := s.questionAgentLocked(agentID)
	id := questionCardID(agentID, callID)
	if recovered := s.findPending(id); recovered != nil && recovered.Continued {
		s.dropPending(id)
	}
	draft := p.drafts[callID]
	if draft == nil {
		draft = newDraftSession(req)
		p.drafts[callID] = draft
	}
	return draft
}
func (s *Store) questionWaterfallDeliveredLocked(pending Pending, outcome dsh.EventResultOutcome) {
	if pending.Event == EventUserQuestions && pending.CallID != "" && outcome.Kind == "result" {
		s.questionAgentLocked(pending.AgentID).answered[pending.CallID] = true
	}
}
func (s *Store) resetQuestionRecoveryLocked() {
	for _, p := range append([]Pending(nil), s.pending...) {
		if p.Continued {
			s.dropPending(p.EventID)
		}
	}
	epoch := s.questionRecovery.epoch + 1
	s.questionRecovery = questionRecovery{epoch: epoch}
}

func (s *Store) resolveContinuedQuestions(eventID string, answer userQuestionsAnswer) {
	s.mu.Lock()
	p := s.findPending(eventID)
	if p == nil || !p.Continued {
		s.mu.Unlock()
		return
	}
	agentID, callID := p.AgentID, p.CallID
	s.mu.Unlock()
	out := api.AskUserQuestionAnswer{Answers: make([]api.AskUserQuestionAnswerAnswersItem, 0, len(answer.Answers))}
	for _, item := range answer.Answers {
		out.Answers = append(out.Answers, api.AskUserQuestionAnswerAnswersItem{Id: item.ID, Selected: item.Selected, Custom: item.Custom})
	}
	s.AnswerUserQuestionsRPC(context.Background(), agentID, callID, out)
}

// submitQuestionAnswer is exclusive with the live waterfall and preserves failed
// submissions for explicit retry. Success is fenced immediately until inbox echo.
func (s *Store) submitQuestionAnswer(ctx context.Context, agentID, callID string, answer api.AskUserQuestionAnswer) {
	agentID, callID = strings.TrimSpace(agentID), strings.TrimSpace(callID)
	id := questionCardID(agentID, callID)
	s.mu.Lock()
	p := s.findPending(id)
	if ctx.Err() != nil || s.client == nil || p == nil || !p.Continued || p.Sending {
		s.mu.Unlock()
		return
	}
	projection := s.questionAgentLocked(agentID)
	if projection.queued[callID] || projection.settled[callID] || projection.answered[callID] {
		s.mu.Unlock()
		return
	}
	if _, submitted := projection.submitted[callID]; submitted {
		s.mu.Unlock()
		return
	}
	s.questionRecovery.nextToken++
	token, epoch, client := s.questionRecovery.nextToken, s.questionRecovery.epoch, s.client
	asOfSeq := max(projection.questionsSeq, projection.inboxSeq)
	if s.questionRecovery.inFlight == nil {
		s.questionRecovery.inFlight = make(map[string]uint64)
	}
	s.questionRecovery.inFlight[id] = token
	p.Sending, p.Error = true, ""
	projection.status = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		var accepted api.AnswerResult
		err := client.Call(ctx, "userQuestions/answer", map[string]any{"agentId": agentID, "callId": callID, "answer": answer}, &accepted)
		s.mu.Lock()
		if s.client != client || s.questionRecovery.epoch != epoch || s.questionRecovery.inFlight[id] != token {
			s.mu.Unlock()
			return
		}
		delete(s.questionRecovery.inFlight, id)
		p = s.findPending(id)
		projection = s.questionAgentLocked(agentID)
		if err != nil {
			if p != nil {
				p.Sending, p.Error = false, err.Error()
			}
		} else if !accepted {
			// A negative RPC result is not an answered projection. Keep the
			// draft and visible failure until the Host's projection settles it.
			if p != nil {
				p.Sending, p.Error = false, "This question is no longer answerable"
			}
		} else {
			projection.submitted[callID] = submittedQuestion{queuedSeen: projection.queued[callID], asOfSeq: asOfSeq}
			projection.status = "submitted"
			s.dropPending(id)
			s.reconcileProjectedQuestionsLocked(agentID)
		}
		s.mu.Unlock()
		s.fire()
	}()
}

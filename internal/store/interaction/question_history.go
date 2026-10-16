package interaction

import (
	"encoding/json"
)

type QuestionHistoryRow struct {
	QuestionRowView
	Selected []string
	Custom   string
}
type QuestionHistoryView struct {
	AgentID, CallID, Title string
	Queued, Open           bool
	Rows                   []QuestionHistoryRow
}
type questionHistoryState struct {
	entries         []QuestionHistoryView
	selected        string
	settled, queued map[string][]userQuestionsAnswerItem
}

// HideQuestion withdraws only the editor. Its waiter, countdown and draft survive.
func (s *Store) HideQuestion(id string)   { s.setQuestionHidden(id, true) }
func (s *Store) ReopenQuestion(id string) { s.setQuestionHidden(id, false) }
func (s *Store) setQuestionHidden(id string, hidden bool) {
	s.mu.Lock()
	p, ds := s.findPending(id), s.questionDrafts[id]
	if p != nil && p.Event == EventUserQuestions && !p.Sending && ds != nil && planReviewOf(ds.req.Questions) == nil {
		ds.hidden = hidden
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ReviewQuestionHistory(agentID, callID string) {
	s.mu.Lock()
	// A live request always wins over an older transcript record for the same call.
	for _, p := range s.pending {
		if p.AgentID == agentID && p.CallID == callID && p.Event == EventUserQuestions {
			if ds := s.questionDrafts[p.EventID]; ds != nil {
				ds.hidden = false
			}
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	if h := s.questionHistory[agentID]; h != nil {
		if callID == "" {
			h.selected = ""
		} else {
			for _, r := range h.entries {
				if r.CallID == callID {
					h.selected = callID
					break
				}
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}

// SyncQuestionHistory is read-only and rebuilds from loaded transcript records.
// Settled/queued answers are accepted only at the sequence already admitted by
// the existing question reducer; it cannot reopen an editable RPC card.
func (s *Store) SyncQuestionHistory(agentID string, events []json.RawMessage, questions, inbox json.RawMessage, qseq, iseq float64) {
	if agentID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.questionHistory == nil {
		s.questionHistory = map[string]*questionHistoryState{}
	}
	h := s.questionHistory[agentID]
	if h == nil {
		h = &questionHistoryState{}
		s.questionHistory[agentID] = h
	}
	p := s.questionRecovery.agents[agentID]
	if p != nil && p.questionsReady && p.questionsSeq == qseq {
		var v struct {
			Active  []json.RawMessage `json:"active"`
			Settled []struct {
				CallID  string          `json:"callId"`
				Answers json.RawMessage `json:"answers"`
			} `json:"settled"`
		}
		if json.Unmarshal(questions, &v) == nil && v.Active != nil {
			h.settled = map[string][]userQuestionsAnswerItem{}
			for _, r := range v.Settled {
				if a, ok := readHistoryAnswers(r.Answers); ok {
					h.settled[r.CallID] = a
				}
			}
		}
	}
	if p != nil && p.inboxReady && p.inboxSeq == iseq {
		var lanes map[string][]struct {
			Source  struct{ Kind, CallID string } `json:"source"`
			Content []struct{ Type, Text string } `json:"content"`
		}
		if json.Unmarshal(inbox, &lanes) == nil {
			h.queued = map[string][]userQuestionsAnswerItem{}
			for _, lane := range []string{"next-step", "next-turn"} {
				for _, r := range lanes[lane] {
					if r.Source.Kind != "user-question-reply" || len(r.Content) == 0 || r.Content[0].Type != "text" {
						continue
					}
					var reply struct {
						Kind, CallID string
						Answers      json.RawMessage
					}
					if json.Unmarshal([]byte(r.Content[0].Text), &reply) == nil && reply.Kind == "answer_to_pending_question" && reply.CallID == r.Source.CallID {
						if a, ok := readHistoryAnswers(reply.Answers); ok {
							h.queued[r.Source.CallID] = a
						}
					}
				}
			}
		}
	}
	calls := map[string][]userQuestionItem{}
	results := map[string][]userQuestionsAnswerItem{}
	order := []string{}
	for _, raw := range events {
		var e struct {
			Type string
			Data struct {
				CallID, SubCallID, Name string
				Arguments               json.RawMessage
				Content                 []struct{ Type, Text string }
				IsError                 bool
				Error                   json.RawMessage
				Message                 struct {
					ToolCallID string
					IsError    bool
					Content    []struct{ Type, Text string }
				}
			}
		}
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		d := e.Data
		if (e.Type == "tool/call" || e.Type == "tool/ptc-dispatch-start" || e.Type == "tool/ptc-dispatch") && d.Name == "ask_user_question" {
			id := d.CallID
			if id == "" {
				id = d.SubCallID
			}
			args := d.Arguments
			var encoded string
			if json.Unmarshal(args, &encoded) == nil {
				args = []byte(encoded)
			}
			if qs := readHistoryQuestions(args); id != "" && qs != nil {
				if _, exists := calls[id]; !exists {
					order = append(order, id)
				}
				calls[id] = qs
			}
		}
		if e.Type == "tool/result" || e.Type == "tool/ptc-dispatch" {
			if d.IsError || d.Message.IsError || (len(d.Error) > 0 && string(d.Error) != "null") {
				continue
			}
			id := d.SubCallID
			content := d.Content
			if e.Type == "tool/result" {
				id = d.Message.ToolCallID
				content = d.Message.Content
			}
			if len(content) != 1 || content[0].Type != "text" {
				continue
			}
			var answer struct{ Answers json.RawMessage }
			if json.Unmarshal([]byte(content[0].Text), &answer) == nil {
				if a, ok := readHistoryAnswers(answer.Answers); ok {
					results[id] = a
				}
			}
		}
	}
	// Previously opened live cards provide question wording until their call page loads.
	if p != nil {
		for id, ds := range p.drafts {
			if calls[id] == nil {
				calls[id] = ds.req.Questions
				order = append(order, id)
			}
		}
	}
	h.entries = nil
	for _, id := range order {
		answers, ok := results[id]
		queued := false
		if p != nil {
			if p.settled[id] {
				answers, ok = h.settled[id]
			} else if p.queued[id] {
				answers, ok = h.queued[id]
				queued = true
			} else {
				for _, a := range p.active {
					if a.CallID == id {
						ok = false
						break
					}
				}
			}
		}
		if !ok {
			continue
		}
		record, valid := pairQuestionHistory(agentID, id, calls[id], answers)
		if !valid {
			continue
		}
		record.Queued = queued
		h.entries = append(h.entries, record)
	}
}
func readHistoryQuestions(raw json.RawMessage) []userQuestionItem {
	var req struct{ Questions []json.RawMessage }
	if json.Unmarshal(raw, &req) != nil || len(req.Questions) == 0 {
		return nil
	}
	out := []userQuestionItem{}
	seen := map[string]bool{}
	for _, r := range req.Questions {
		var q userQuestionItem
		var required struct {
			ID, Question *string
			MultiSelect  bool `json:"multi_select"`
		}
		if json.Unmarshal(r, &q) != nil || json.Unmarshal(r, &required) != nil || required.ID == nil || required.Question == nil || q.ID == "" || seen[q.ID] {
			return nil
		}
		seen[q.ID] = true
		q.MultiSelect = q.MultiSelect || required.MultiSelect
		out = append(out, q)
	}
	return out
}
func readHistoryAnswers(raw json.RawMessage) ([]userQuestionsAnswerItem, bool) {
	var a []userQuestionsAnswerItem
	if json.Unmarshal(raw, &a) != nil || a == nil {
		return nil, false
	}
	seen := map[string]bool{}
	for _, r := range a {
		if r.ID == "" || r.Selected == nil || seen[r.ID] {
			return nil, false
		}
		seen[r.ID] = true
	}
	return a, true
}
func pairQuestionHistory(agentID, id string, questions []userQuestionItem, answers []userQuestionsAnswerItem) (QuestionHistoryView, bool) {
	v := QuestionHistoryView{AgentID: agentID, CallID: id, Title: questionCardTitle(userQuestionsRequest{Questions: questions})}
	if len(questions) == 0 || len(questions) != len(answers) {
		return v, false
	}
	byID := map[string]userQuestionsAnswerItem{}
	for _, a := range answers {
		if _, ok := byID[a.ID]; ok {
			return v, false
		}
		byID[a.ID] = a
	}
	seen := map[string]bool{}
	for _, q := range questions {
		a, ok := byID[q.ID]
		if !ok || seen[q.ID] {
			return v, false
		}
		seen[q.ID] = true
		row := QuestionHistoryRow{QuestionRowView: QuestionRowView{ID: q.ID, Question: q.Question, Detail: q.Detail, Header: q.Header, MultiSelect: q.MultiSelect}, Selected: append([]string(nil), a.Selected...)}
		if a.Custom != nil {
			row.Custom = *a.Custom
		}
		row.Skipped = len(a.Selected) == 0 && row.Custom == ""
		for _, o := range q.Options {
			row.OptionLabels = append(row.OptionLabels, o.Label)
			row.OptionDescriptions = append(row.OptionDescriptions, o.Description)
		}
		v.Rows = append(v.Rows, row)
	}
	return v, true
}
func (s *Store) questionHistorySnapshotLocked(agentID string) []QuestionHistoryView {
	h := s.questionHistory[agentID]
	if h == nil {
		return nil
	}
	out := make([]QuestionHistoryView, 0, len(h.entries))
	for _, v := range h.entries {
		live := false
		for _, p := range s.pending {
			if p.AgentID == agentID && p.CallID == v.CallID && p.Event == EventUserQuestions {
				live = true
				break
			}
		}
		if live {
			continue
		}
		v.Open = v.CallID == h.selected
		v.Rows = append([]QuestionHistoryRow(nil), v.Rows...)
		for i := range v.Rows {
			r := &v.Rows[i]
			r.Selected = append([]string(nil), r.Selected...)
			r.OptionLabels = append([]string(nil), r.OptionLabels...)
			r.OptionDescriptions = append([]string(nil), r.OptionDescriptions...)
		}
		out = append(out, v)
	}
	return out
}

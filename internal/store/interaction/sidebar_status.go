package interaction

import (
	"encoding/json"
	"sync"
)

func PendingStatusRank(status string) int {
	switch status {
	case "plan-review":
		return 3
	case "question":
		return 2
	case "approval":
		return 1
	}
	return 0
}
func preferPending(statuses map[string]string, id, status string) {
	if id != "" && PendingStatusRank(status) > PendingStatusRank(statuses[id]) {
		statuses[id] = status
	}
}

// PendingStatuses is passive presentation only. Resolved approvals and already
// submitted cards do not advertise another required decision.
func (s *Store) PendingStatuses() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for _, p := range s.pending {
		if p.Sending || p.approvalResolved {
			continue
		}
		switch p.Event {
		case EventApproval:
			preferPending(out, p.AgentID, "approval")
		case EventUserQuestions:
			kind := "question"
			if d := s.questionDrafts[p.EventID]; d != nil && planReviewOf(d.req.Questions) != nil {
				kind = "plan-review"
			}
			preferPending(out, p.AgentID, kind)
		}
	}
	return out
}

// ProjectedPendingStatus reads durable active questions without registering a
// waterfall or manufacturing answer authority. A continued question needs a
// known inbox with no queued reply. Unknown/malformed faces add no status.
func ProjectedPendingStatus(questions, inbox json.RawMessage) string {
	projectedStatusMemo.Lock()
	if status, ok := projectedStatusMemo.byQuestions[string(questions)][string(inbox)]; ok {
		projectedStatusMemo.Unlock()
		return status
	}
	projectedStatusMemo.Unlock()
	status := projectedPendingStatus(questions, inbox, nil)
	projectedStatusMemo.Lock()
	if projectedStatusMemo.byQuestions == nil || projectedStatusMemo.size >= projectedStatusMemoLimit {
		projectedStatusMemo.byQuestions, projectedStatusMemo.size = map[string]map[string]string{}, 0
	}
	inner := projectedStatusMemo.byQuestions[string(questions)]
	if inner == nil {
		inner = map[string]string{}
		projectedStatusMemo.byQuestions[string(questions)] = inner
	}
	inner[string(inbox)] = status
	projectedStatusMemo.size++
	projectedStatusMemo.Unlock()
	return status
}

// projectedStatusMemo remembers ProjectedPendingStatus per exact input. The
// sidebar asks for every session's status on every frame while the projected
// documents rarely change, and decoding them each time showed in hover frames.
var projectedStatusMemo struct {
	sync.Mutex
	byQuestions map[string]map[string]string
	size        int
}

const projectedStatusMemoLimit = 1024

// projectedPendingStatus is a pure function of its inputs.
func projectedPendingStatus(questions, inbox json.RawMessage, ignored map[string]bool) string {
	var face struct {
		Active  []projectedQuestion `json:"active"`
		Settled []struct {
			CallID string `json:"callId"`
		} `json:"settled"`
	}
	if json.Unmarshal(questions, &face) != nil || face.Active == nil {
		return ""
	}
	settled := map[string]bool{}
	for _, r := range face.Settled {
		settled[r.CallID] = true
	}
	queued := map[string]bool{}
	var lanes map[string]json.RawMessage
	inboxKnown := json.Unmarshal(inbox, &lanes) == nil && lanes != nil
	found := false
	for _, key := range []string{"next-step", "next-turn"} {
		raw, ok := lanes[key]
		if !ok {
			continue
		}
		found = true
		var rows []struct {
			Source struct{ Kind, CallID string } `json:"source"`
		}
		if json.Unmarshal(raw, &rows) != nil {
			inboxKnown = false
			break
		}
		for _, r := range rows {
			if r.Source.Kind == "user-question-reply" {
				queued[r.Source.CallID] = true
			}
		}
	}
	inboxKnown = inboxKnown && found
	result := ""
	for _, r := range face.Active {
		if ignored[r.CallID] || r.CallID == "" || len(r.Questions) == 0 || settled[r.CallID] || queued[r.CallID] {
			continue
		}
		if r.State != "open" && (r.State != "continued" || !inboxKnown) {
			continue
		}
		kind := "question"
		if planReviewOf(r.Questions) != nil {
			kind = "plan-review"
		}
		if PendingStatusRank(kind) > PendingStatusRank(result) {
			result = kind
		}
	}
	return result
}

func (s *Store) ProjectedSidebarStatus(agentID string, questions, inbox json.RawMessage) string {
	s.mu.Lock()
	ignored := map[string]bool{}
	if p := s.questionRecovery.agents[agentID]; p != nil {
		for id, answered := range p.answered {
			if answered {
				ignored[id] = true
			}
		}
		for id := range p.submitted {
			ignored[id] = true
		}
	}
	for _, p := range s.pending {
		if p.AgentID == agentID && p.Sending && p.CallID != "" {
			ignored[p.CallID] = true
		}
	}
	s.mu.Unlock()
	return projectedPendingStatus(questions, inbox, ignored)
}

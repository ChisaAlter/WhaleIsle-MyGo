package session

import (
	"encoding/json"
	"whaleisle/internal/workflowrun"
)

const MessageKindWorkflow MessageKind = "workflow"

func (s *Store) captureWorkflowBoundaryLocked(raw json.RawMessage) {
	if !workflowrun.Boundary(raw) {
		return
	}
	var e struct {
		Seq float64 `json:"seq"`
	}
	if json.Unmarshal(raw, &e) != nil {
		return
	}
	if s.workflowBoundaries == nil {
		s.workflowBoundaries = map[float64]json.RawMessage{}
	}
	s.workflowBoundaries[e.Seq] = append(json.RawMessage(nil), raw...)
}
func (s *Store) workflowRunsLocked(messages []Message) []workflowrun.Run {
	raws := make([]json.RawMessage, 0, len(messages)+len(s.workflowBoundaries))
	for _, raw := range s.workflowBoundaries {
		raws = append(raws, raw)
	}
	for _, m := range messages {
		if len(m.Raw) > 0 {
			raws = append(raws, m.Raw)
		}
	}
	return workflowrun.Fold(raws)
}

// ProjectWorkflowMessages is chat-only: trajectory, source journal, edit/fork
// anchors and paging retain the original durable workflow records.
func ProjectWorkflowMessages(messages []Message, runs []workflowrun.Run) []Message {
	anchors := map[float64]*workflowrun.Run{}
	owned := map[float64]bool{}
	for i := range runs {
		r := &runs[i]
		anchors[r.AnchorSeq] = r
		for _, seq := range r.EventSeqs {
			owned[seq] = true
		}
	}
	out := make([]Message, 0, len(messages))
	for _, m := range messages {
		if r := anchors[m.Seq]; r != nil {
			m.Kind = MessageKindWorkflow
			m.Workflow = r
			m.ID = "workflow:" + r.ID
			m.Body = r.Name
			out = append(out, m)
		} else if !owned[m.Seq] {
			out = append(out, m)
		}
	}
	return out
}
func WorkflowMemberAvailable(items []Item, parent string, member workflowrun.Member) bool {
	_, available := workflowMemberMode(items, parent, member)
	return available
}

func workflowMemberMode(items []Item, parent string, member workflowrun.Member) (string, bool) {
	if member.Status != "running" || parent == "" || member.ChildID == "" {
		return "", false
	}
	for _, it := range items {
		if it.SessionID == member.ChildID && it.ParentID == parent && it.Running && it.SubagentIdentity != nil {
			mode := it.SubagentIdentity.Mode
			if mode == "one-shot" || mode == "continuable" {
				return mode, true
			}
		}
	}
	return "", false
}
func (s *Store) workflowMemberAllowedLocked(parent, runID string, seq float64, child string) bool {
	_, available := s.workflowMemberAddressLocked(parent, runID, seq, child)
	return available
}

func (s *Store) workflowMemberAddressLocked(parent, runID string, seq float64, child string) (map[string]any, bool) {
	if s.selectedID != parent {
		return nil, false
	}
	runs := s.workflowRunsLocked(projectMessageEdits(s.messages))
	for _, r := range runs {
		if r.ID != runID {
			continue
		}
		for _, p := range r.Phases {
			for _, m := range p.Members {
				if m.Seq == seq && m.ChildID == child {
					mode, available := workflowMemberMode(s.items, parent, m)
					if available {
						return map[string]any{"kind": "subagent", "parentSessionId": parent, "childSessionId": child, "mode": mode}, true
					}
					return nil, false
				}
			}
		}
	}
	return nil, false
}

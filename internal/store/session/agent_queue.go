package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/dsh/api"
)

func seedChildQueue(v *AgentNavigationView, values json.RawMessage, seq ...*float64) {
	sequenced := len(seq) > 0 && validSidebarCatalogSeq(seq[0])
	if sequenced && v.queueInboxSeeded && *seq[0] <= v.queueInboxSeq {
		return
	}
	var data struct {
		Inbox struct {
			Next []struct {
				ID      string              `json:"id"`
				Content []queueContentBlock `json:"content"`
				Source  struct {
					Kind string `json:"kind"`
				}
			} `json:"next-turn"`
		} `json:"inbox"`
	}
	if json.Unmarshal(values, &data) != nil || data.Inbox.Next == nil {
		return
	}
	if sequenced {
		v.queueInboxSeq, v.queueInboxSeeded = *seq[0], true
	}
	rows := make([]PlanQueueItem, 0, len(data.Inbox.Next))
	for _, row := range data.Inbox.Next {
		item := PlanQueueItem{ID: row.ID, Source: row.Source.Kind, ContentKnown: true, TextOnly: true}
		var text []string
		for _, part := range row.Content {
			if part.Type == "text" {
				text = append(text, part.Text)
			} else {
				item.TextOnly = false
				if part.Attachment != nil {
					a := *part.Attachment
					a.Kind = part.Type
					item.Attachments = append(item.Attachments, a)
				}
			}
		}
		item.Preview = truncateRunes(strings.Join(text, " "), 120)
		rows = append(rows, item)
	}
	v.Board.Queue = rows
	if v.QueueEditor.ItemID != "" {
		present := false
		for _, row := range rows {
			present = present || row.ID == v.QueueEditor.ItemID
		}
		if !present {
			v.QueueEditor = QueueEditorState{Token: v.QueueEditor.Token + 1}
		}
	}
}

// The addressed follow carries journal events, while the shared control reader
// carries authoritative inbox values for every Session. Reuse that exact child
// projection instead of reconstructing pending content from journal previews.
func (s *Store) syncChildQueueProjectionLocked(v *AgentNavigationView) {
	if !v.Address.validChild() || !v.Process.Live {
		return
	}
	p := s.questionProjections[v.Address.ChildSessionID]
	if owner := s.agentNavigationOwner; owner != nil {
		owner.mu.Lock()
		other := owner.questionProjections[v.Address.ChildSessionID]
		if other.Inbox != nil && (p.Inbox == nil || other.InboxSeq >= p.InboxSeq) {
			p.Inbox, p.InboxSeq = other.Inbox, other.InboxSeq
		}
		owner.mu.Unlock()
	}
	if p.Inbox == nil || v.queueInboxSeeded && p.InboxSeq <= v.queueInboxSeq {
		return
	}
	values, _ := json.Marshal(map[string]json.RawMessage{"inbox": p.Inbox})
	seedChildQueue(v, values, &p.InboxSeq)
}

func childQueueFrame(v *AgentNavigationView, raw json.RawMessage) {
	var frame struct {
		Type, Key string
		Value     json.RawMessage
		Event     json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	if frame.Type == "projection" && frame.Key == "inbox" {
		values, _ := json.Marshal(map[string]json.RawMessage{"inbox": frame.Value})
		seedChildQueue(v, values)
	}
	for _, record := range v.Process.Records {
		if record.Event.Type == "turn/start" {
			v.Running = true
		}
		if record.Event.Type == "turn/end" {
			v.Running = false
		}
	}
	// Process records include non-transcript boundaries for the active turn.
	// Presented messages can end at an older turn/end while the next turn is
	// waiting for its first assistant message; they must not overwrite that state.
}
func (s *Store) childQueueViewLocked(v *AgentNavigationView) {
	v.Board.QueueActions = QueueActionState{Scope: fmt.Sprintf("%s:%d:%s", s.agentNavigation.binding, v.Token, v.Address.ChildSessionID), Mutable: s.agentPromptActionAdmissionLocked(v.Token, v.Address, true) && s.agentParentCanPromptLocked(v.Address) && v.Process.Live, Running: v.Running, Busy: v.ActionBusy || v.QueueEditor.Loading || v.QueueEditor.Saving}
}
func (s *Store) BeginAgentQueueEdit(token uint64, address AgentSessionAddress, id string) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentPromptActionAdmissionLocked(token, address, true) || !s.agentParentCanPromptLocked(address) || !n.view.Process.Live || n.view.ActionBusy || id == "" {
		s.mu.Unlock()
		return
	}
	n.view.QueueEditor = QueueEditorState{ItemID: id, Token: n.view.QueueEditor.Token + 1, Loading: true}
	e := n.view.QueueEditor
	client := n.client
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var out api.SessionProjectionsValue
		err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": address.ChildSessionID}}, &out)
		if err == nil {
			err = out.RequireSequenced()
		}
		text := ""
		if err == nil {
			text, err = queueEditableText(out.Values, id)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.agentPromptActionAdmissionLocked(token, address, true) || s.agentNavigation.client != client || s.agentNavigation.view.QueueEditor.Token != e.Token {
			return
		}
		edit := &s.agentNavigation.view.QueueEditor
		edit.Loading = false
		if err != nil {
			edit.Error = err.Error()
		} else {
			edit.Text = text
			edit.Ready = true
		}
		go s.fire()
	}()
}
func (s *Store) ChangeAgentQueueEdit(token, edit uint64, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := &s.agentNavigation.view
	if v.Token == token && v.QueueEditor.Token == edit && v.QueueEditor.Ready && !v.QueueEditor.Saving {
		v.QueueEditor.Text = text
		v.QueueEditor.Error = ""
	}
}
func (s *Store) CancelAgentQueueEdit(token, edit uint64) {
	s.mu.Lock()
	v := &s.agentNavigation.view
	if v.Token == token && v.QueueEditor.Token == edit && !v.QueueEditor.Saving {
		v.QueueEditor = QueueEditorState{Token: edit + 1}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SaveAgentQueueEdit(token uint64, address AgentSessionAddress, edit uint64) {
	s.mu.Lock()
	e := s.agentNavigation.view.QueueEditor
	s.mu.Unlock()
	if e.Token != edit || !e.Ready || e.Saving {
		return
	}
	s.UpdateAgentQueue(token, address, e.ItemID, QueueActionEdit, e.Text)
}
func (s *Store) UpdateAgentQueue(token uint64, address AgentSessionAddress, id string, action UpdateQueueAction, text ...string) {
	payload, ok := queueActionPayload(action, text)
	if !ok {
		return
	}
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentPromptActionAdmissionLocked(token, address, true) || !s.agentParentCanPromptLocked(address) || n.view.ActionBusy || !n.view.Process.Live {
		s.mu.Unlock()
		return
	}
	n.view.ActionBusy = true
	client := n.client
	n.actionEpoch++
	epoch := n.actionEpoch
	if action == QueueActionEdit {
		n.view.QueueEditor.Saving = true
	}
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var out struct {
			Accepted bool `json:"accepted"`
		}
		err := client.Call(ctx, "session/updateQueue", map[string]any{"request": map[string]any{"sessionId": address.ChildSessionID, "itemId": id, "action": payload}}, &out)
		if err == nil && !out.Accepted {
			err = errors.New("queue mutation was not accepted")
		}
		s.mu.Lock()
		n := &s.agentNavigation
		if !s.agentPromptActionAdmissionLocked(token, address, true) || n.client != client || n.actionEpoch != epoch {
			s.mu.Unlock()
			return
		}
		n.view.ActionBusy = false
		n.view.QueueEditor.Saving = false
		if err != nil && !(action == QueueActionSteer && queueSteerConverged(err)) {
			n.view.ActionStatus = err.Error()
			if action == QueueActionEdit {
				n.view.QueueEditor.Error = err.Error()
			}
		} else {
			next := n.view.Board.Queue[:0]
			for _, row := range n.view.Board.Queue {
				if row.ID == id {
					if action != QueueActionEdit {
						continue
					}
					row.Preview = truncateRunes(text[0], 120)
				}
				next = append(next, row)
			}
			n.view.Board.Queue = next
			if action == QueueActionEdit {
				n.view.QueueEditor = QueueEditorState{Token: n.view.QueueEditor.Token + 1}
			}
			n.view.ActionStatus = ""
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Steer all captures one current child inbox and converges each occurrence. It
// never turns an empty accelerated gesture into a blank model prompt.
func (s *Store) SteerAgentQueue(token uint64, address AgentSessionAddress) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentPromptActionAdmissionLocked(token, address, true) || !n.view.Running || n.view.ActionBusy || !n.view.Process.Live {
		s.mu.Unlock()
		return
	}
	rows := append([]PlanQueueItem(nil), n.view.Board.Queue...)
	client := n.client
	n.view.ActionBusy = true
	n.actionEpoch++
	epoch := n.actionEpoch
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var last error
		for _, row := range rows {
			s.mu.Lock()
			valid := s.agentPromptActionAdmissionLocked(token, address, true) && s.agentNavigation.client == client && s.agentNavigation.actionEpoch == epoch
			s.mu.Unlock()
			if !valid {
				return
			}
			var out struct {
				Accepted bool `json:"accepted"`
			}
			err := client.Call(ctx, "session/updateQueue", map[string]any{"request": map[string]any{"sessionId": address.ChildSessionID, "itemId": row.ID, "action": map[string]any{"kind": "steer"}}}, &out)
			if err == nil && !out.Accepted {
				err = errors.New("queue mutation was not accepted")
			}
			if err != nil && !queueSteerConverged(err) {
				last = err
				break
			}
			s.mu.Lock()
			if s.agentPromptActionAdmissionLocked(token, address, true) && s.agentNavigation.client == client && s.agentNavigation.actionEpoch == epoch {
				q := s.agentNavigation.view.Board.Queue
				next := q[:0]
				for _, item := range q {
					if item.ID != row.ID {
						next = append(next, item)
					}
				}
				s.agentNavigation.view.Board.Queue = next
			}
			s.mu.Unlock()
		}
		s.mu.Lock()
		if s.agentPromptActionAdmissionLocked(token, address, true) && s.agentNavigation.client == client && s.agentNavigation.actionEpoch == epoch {
			s.agentNavigation.view.ActionBusy = false
			if last != nil {
				s.agentNavigation.view.ActionStatus = last.Error()
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

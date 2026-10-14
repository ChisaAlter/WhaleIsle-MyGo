package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// QueueEditorState is a session-scoped draft, separate from the send composer.
type QueueEditorState struct {
	ItemID, Text, Error    string
	Loading, Saving, Ready bool
	Token                  uint64
}
type queueEditor struct {
	QueueEditorState
	sessionID  string
	generation uint64
}

func (s *Store) queueEditorSnapshotLocked() QueueEditorState {
	if s.queueEditor.sessionID != s.selectedID || s.queueEditor.generation != s.followGeneration {
		return QueueEditorState{}
	}
	return s.queueEditor.QueueEditorState
}
func (s *Store) queueEditorCurrentLocked(ctx context.Context, client *dsh.Client, token uint64) bool {
	e := s.queueEditor
	return e.Token == token && s.sessionResultCurrentLocked(ctx, client, e.sessionID, e.generation)
}

// BeginQueueEdit reads current durable content. The displayed preview is never
// an editing source; mixed attachment content remains unchanged and uneditable.
func (s *Store) BeginQueueEdit(ctx context.Context, itemID string) {
	s.mu.Lock()
	if s.client == nil || s.selectedID == "" || itemID == "" || s.queueEditorSnapshotLocked().Saving {
		s.mu.Unlock()
		return
	}
	s.queueEditorToken++
	s.queueEditor = queueEditor{QueueEditorState: QueueEditorState{ItemID: itemID, Loading: true, Token: s.queueEditorToken}, sessionID: s.selectedID, generation: s.followGeneration}
	client, token, sid := s.client, s.queueEditorToken, s.selectedID
	s.mu.Unlock()
	s.fire()
	go s.runLoadQueueEdit(ctx, client, sid, token, itemID)
}
func (s *Store) runLoadQueueEdit(ctx context.Context, client *dsh.Client, sid string, token uint64, itemID string) {
	var out api.SessionProjectionsValue
	err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": sid}}, &out)
	var text string
	if err == nil {
		text, err = queueEditableText(out.Values, itemID)
	}
	s.mu.Lock()
	if !s.queueEditorCurrentLocked(ctx, client, token) {
		s.mu.Unlock()
		return
	}
	s.queueEditor.Loading = false
	if err != nil {
		s.queueEditor.Error = err.Error()
	} else {
		s.queueEditor.Text = text
		s.queueEditor.Ready = true
	}
	s.mu.Unlock()
	s.fire()
}
func queueEditableText(values json.RawMessage, itemID string) (string, error) {
	var data struct {
		Inbox struct {
			Next []struct {
				ID      string `json:"id"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"next-turn"`
		} `json:"inbox"`
	}
	if json.Unmarshal(values, &data) != nil {
		return "", errors.New("queue: could not read current content")
	}
	for _, row := range data.Inbox.Next {
		if row.ID != itemID {
			continue
		}
		var text strings.Builder
		for _, part := range row.Content {
			if part.Type != "text" {
				return "", errors.New("queue: messages with attachments cannot be edited")
			}
			text.WriteString(part.Text)
		}
		return text.String(), nil
	}
	return "", errors.New("queue: item is no longer pending")
}
func (s *Store) ChangeQueueEdit(token uint64, text string) {
	s.mu.Lock()
	if e := s.queueEditorSnapshotLocked(); e.Token == token && e.Ready && !e.Saving {
		s.queueEditor.Text = text
		s.queueEditor.Error = ""
	}
	s.mu.Unlock()
}
func (s *Store) CancelQueueEdit(token uint64) {
	s.mu.Lock()
	if e := s.queueEditorSnapshotLocked(); e.Token == token && !e.Saving {
		s.queueEditorToken++
		s.queueEditor = queueEditor{}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SaveQueueEdit(ctx context.Context, token uint64) {
	s.mu.Lock()
	e := s.queueEditorSnapshotLocked()
	if e.Token != token || !e.Ready || e.Saving || strings.TrimSpace(e.Text) == "" {
		s.mu.Unlock()
		return
	}
	s.queueEditor.Saving = true
	s.queueEditor.Error = ""
	client, sid := s.client, s.selectedID
	s.mu.Unlock()
	s.fire()
	go s.runSaveQueueEdit(ctx, client, sid, e)
}
func (s *Store) runSaveQueueEdit(ctx context.Context, client *dsh.Client, sid string, edit QueueEditorState) {
	action, _ := queueActionPayload(QueueActionEdit, []string{edit.Text})
	var out struct {
		Accepted bool `json:"accepted"`
	}
	err := client.Call(ctx, "session/updateQueue", map[string]any{"request": map[string]any{"sessionId": sid, "itemId": edit.ItemID, "action": action}}, &out)
	if err == nil && !out.Accepted {
		err = errors.New("queue: mutation was not accepted")
	}
	s.mu.Lock()
	if !s.queueEditorCurrentLocked(ctx, client, edit.Token) {
		s.mu.Unlock()
		return
	}
	s.queueEditor.Saving = false
	if err != nil {
		s.queueEditor.Error = err.Error()
	} else {
		for i := range s.planBoard.Queue {
			if s.planBoard.Queue[i].ID == edit.ItemID {
				s.planBoard.Queue[i].Preview = truncateRunes(edit.Text, 120)
			}
		}
		s.lastAction = "queue edit " + edit.ItemID
		s.queueEditor = queueEditor{}
	}
	s.mu.Unlock()
	s.fire()
}

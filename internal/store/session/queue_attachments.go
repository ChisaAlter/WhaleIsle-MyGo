package session

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"whaleisle/internal/dsh/api"
)

// QueueAttachment is a durable Host reference, not a path or staged receipt.
type QueueAttachment struct {
	Kind         string  `json:"-"`
	AttachmentID string  `json:"attachmentId"`
	Name         string  `json:"name"`
	MediaType    string  `json:"mediaType"`
	Bytes        float64 `json:"bytes"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
}
type queueContentBlock struct {
	Type       string           `json:"type"`
	Text       string           `json:"text"`
	Attachment *QueueAttachment `json:"attachment"`
}

// QueueActionState applies to the normal selected-session address. It does not
// infer read-only status from Origin: continuable child queues remain valid.
type QueueActionState struct {
	Scope                  string
	Mutable, Running, Busy bool
}

func (a QueueActionState) Can(action UpdateQueueAction, item PlanQueueItem) bool {
	if !a.Mutable || a.Busy || item.ID == "" {
		return false
	}
	switch action {
	case QueueActionSteer:
		return a.Running
	case QueueActionEdit:
		return !item.ContentKnown || item.TextOnly
	case QueueActionRemove:
		return true
	}
	return false
}
func (s *Store) queueActionsLocked() QueueActionState {
	a := QueueActionState{Scope: fmt.Sprintf("%p:%d:%s", s.client, s.followGeneration, s.selectedID), Mutable: s.client != nil && s.selectedID != "", Running: s.turnStartMs > 0}
	for _, item := range s.items {
		if item.SessionID == s.selectedID && item.Running {
			a.Running = true
		}
	}
	e := s.queueEditorSnapshotLocked()
	a.Busy = e.Loading || e.Saving || s.queueMutationScope == a.Scope && s.queueMutationID != ""
	return a
}
func (s *Store) queueItemLocked(id string) PlanQueueItem {
	for _, item := range s.planBoard.Queue {
		if item.ID == id {
			return item
		}
	}
	// A historical preview may arrive before its inbox projection. Let the Host
	// validate that row; do not mistake missing local metadata for no capability.
	return PlanQueueItem{ID: id}
}

// QueueImageLoader captures the exact render owner. Both request and result
// are guarded; attachment IDs are never reused across sessions or Host epochs.
func (s *Store) QueueImageLoader(sessionID, scope string) func(context.Context, string) (string, error) {
	return func(ctx context.Context, id string) (string, error) {
		s.mu.Lock()
		client := s.client
		valid := s.selectedID == sessionID && s.queueActionsLocked().Scope == scope && id != ""
		s.mu.Unlock()
		if !valid || client == nil {
			return "", fmt.Errorf("queue image owner changed")
		}
		var out api.SessionAttachmentValue
		err := client.Call(ctx, "session/attachment", map[string]any{"request": api.SessionAttachmentRequest{SessionId: sessionID, AttachmentId: id}}, &out)
		if err != nil {
			return "", err
		}
		if out.Attachment.AttachmentId != id {
			return "", fmt.Errorf("queue image identity mismatch")
		}
		raw, err := decodeAttachmentData(out.Data)
		if err != nil {
			return "", err
		}
		s.mu.Lock()
		valid = s.client == client && s.selectedID == sessionID && s.queueActionsLocked().Scope == scope && ctx.Err() == nil
		s.mu.Unlock()
		if !valid {
			return "", fmt.Errorf("queue image owner changed")
		}
		attachmentCache.mu.Lock()
		dir := attachmentCache.dir
		attachmentCache.mu.Unlock()
		if dir == "" {
			dir = filepath.Join(os.TempDir(), "whaleisle-attachments")
		}
		if err = os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		key := sha256.Sum256([]byte(scope + "\x00" + id))
		path := filepath.Join(dir, fmt.Sprintf("queue-%x.image", key))
		if err = os.WriteFile(path, raw, 0600); err != nil {
			return "", err
		}
		return path, nil
	}
}

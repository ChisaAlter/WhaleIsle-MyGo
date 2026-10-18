package previewprocess

import (
	"crypto/sha256"
	"encoding/json"
	"errors"

	"whaleisle/internal/desktop/openui"
)

func mediaRequestHash(e openui.Event) [sha256.Size]byte {
	raw, _ := json.Marshal(e)
	return sha256.Sum256(raw)
}

// ClaimOpenUIMedia consumes exactly the typed event admitted from this owned,
// inline viewer. It never creates a window. The chat owner must separately
// recheck source revision, component activation, visibility and native clip.
func (s *Session) ClaimOpenUIMedia(e openui.Event) error {
	if s == nil || s.openUIEvents == nil || s.openUIAPI == nil {
		return errors.New("OpenUI media transport is unavailable")
	}
	if err := openui.ValidateMediaRequest(e); err != nil {
		return err
	}
	s.openUIAPI.mu.Lock()
	canReply := s.openUIAPI.capable && !s.openUIAPI.closed
	s.openUIAPI.mu.Unlock()
	if !canReply {
		return errors.New("preview helper cannot acknowledge media requests")
	}
	q := s.openUIEvents
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || !q.media || e.InstanceID != q.instanceID {
		return errors.New("OpenUI media owner is unavailable")
	}
	want, exists := q.pendingMedia[e.RequestID]
	if !exists || want != mediaRequestHash(e) {
		return errors.New("unknown or consumed OpenUI media request")
	}
	delete(q.pendingMedia, e.RequestID)
	return nil
}

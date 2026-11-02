package previewprocess

import (
	"crypto/sha256"
	"encoding/json"
	"errors"

	"whaleisle/internal/desktop/openui"
)

type openUIActionKey struct {
	digest [sha256.Size]byte
}

func nativeActionKey(e openui.Event) openUIActionKey {
	// Download content arrives as separately validated ordered parts; the final
	// one-use ticket binds the request identity, filename, MIME and part count.
	if e.Type == "download" {
		e.Content = ""
	}
	raw, _ := json.Marshal(e)
	return openUIActionKey{sha256.Sum256(raw)}
}

// ClaimOpenUINativeAction consumes exactly one action received from this
// session's owned viewer. It never performs an action or attests a gesture.
// The caller must check the original UI owner immediately before the native
// call. Repeated claims, forged payloads, closed sessions and unavailable reply
// transports fail without executing anything.
func (s *Session) ClaimOpenUINativeAction(e openui.Event) error {
	if s == nil || s.openUIEvents == nil || s.openUIAPI == nil {
		return errors.New("OpenUI native actions are unavailable")
	}
	if err := openui.ValidateNativeAction(e); err != nil {
		return err
	}
	s.openUIAPI.mu.Lock()
	canReply := s.openUIAPI.capable && !s.openUIAPI.closed
	s.openUIAPI.mu.Unlock()
	if !canReply {
		return errors.New("preview helper cannot acknowledge native actions")
	}
	q := s.openUIEvents
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || !q.nativeActions || e.InstanceID != q.instanceID {
		return errors.New("OpenUI native action owner is unavailable")
	}
	key, pending := q.pendingActions[e.RequestID]
	if !pending || key != nativeActionKey(e) {
		return errors.New("unknown or consumed OpenUI native action")
	}
	delete(q.pendingActions, e.RequestID)
	return nil
}

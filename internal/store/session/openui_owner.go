package session

import (
	"errors"
	"strings"
)

// CaptureOpenUI pins the user-visible assistant block's queue target and returns
// a revocation check. It never derives an address from renderer event fields.
func (s *Store) CaptureOpenUI(binding, sid, messageID, program string) (*ComposerTarget, func() bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	contains := func() bool {
		for _, m := range s.messages {
			if (m.ID == messageID || m.SourceMessageID == messageID) && m.Role == "assistant" && strings.Contains(m.Body, program) {
				return true
			}
		}
		return false
	}
	if s.client == nil || sid == "" || s.selectedID != sid || s.planHistoryBinding != binding || program == "" || !contains() {
		return nil, nil, errors.New("OpenUI block owner is no longer active")
	}
	target := &ComposerTarget{SessionID: sid, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration, mode: PromptModeQueue}
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return target.currentLocked() && s.followGeneration == target.followGeneration && s.selectedID == sid && s.planHistoryBinding == binding && contains()
	}
	return target, valid, nil
}

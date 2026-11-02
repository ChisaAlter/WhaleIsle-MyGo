package session

import (
	"context"
	"strings"
)

// PickComposerCommand captures the rendered Session, Host connection and epoch
// before detaching from the menu. A later navigation can neither redirect it
// nor apply an old result to the newly selected Session.
func (s *Store) PickComposerCommand(hostGeneration uint64, sessionID, line string) {
	s.mu.Lock()
	if s.client == nil || s.composerHostGeneration != hostGeneration || s.selectedID != sessionID || sessionID == "" {
		s.mu.Unlock()
		return
	}
	catalog := &s.commandCatalog
	if catalog.client != s.client || catalog.sessionID != sessionID || !catalog.loaded || catalog.err != nil {
		s.mu.Unlock()
		return
	}
	descriptor, ok := commandForLine(catalog.items, strings.TrimSpace(line))
	if !ok || descriptor.Input != nil {
		s.mu.Unlock()
		return
	}
	target := &ComposerTarget{SessionID: sessionID, store: s, client: s.client, epoch: s.composerHostGeneration, followGeneration: s.followGeneration}
	s.mu.Unlock()
	go func() {
		ctx := context.Background()
		if target.check(ctx) != nil {
			return
		}
		err := executeCommand(ctx, target.client, target.SessionID, line, nil)
		s.mu.Lock()
		current := target.currentLocked() && s.selectedID == target.SessionID && s.followGeneration == target.followGeneration
		if current && err != nil {
			if s.lastAction != err.Error() {
				s.invalidateViewLocked()
			}
			s.lastAction = err.Error()
		}
		s.mu.Unlock()
		if current && err != nil {
			s.fire()
		}
	}()
}

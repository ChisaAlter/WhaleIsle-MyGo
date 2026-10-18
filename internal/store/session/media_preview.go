package session

import (
	"errors"
	"whaleisle/internal/desktop/mediapreview"
)

// MediaPreviewSource captures the authenticated connection and selection epoch.
// The resulting reader can only use published read-only workspaceFiles methods;
// it is never exposed to the player or its native bridge.
func (s *Store) MediaPreviewSource(sid string) (mediapreview.Caller, func() bool, error) {
	s.mu.Lock()
	client, generation := s.client, s.followGeneration
	current := client != nil && sid != "" && s.selectedID == sid
	s.mu.Unlock()
	if !current {
		return nil, nil, errors.New("media preview requires the selected session's Host file reader")
	}
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.client == client && s.selectedID == sid && s.followGeneration == generation
	}
	return client, valid, nil
}

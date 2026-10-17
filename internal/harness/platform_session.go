package harness

import (
	"context"
	"errors"
	"whaleisle/internal/desktop/platformview"
)

// PlatformSession never exposes the publisher credential or account token in
// Status/logging. A restarted Host cannot authorize an old document open.
func (s *Supervisor) PlatformSession(ctx context.Context) (platformview.Session, error) {
	s.mu.Lock()
	generation, state, token := s.gen, s.status, s.platformToken
	s.mu.Unlock()
	if state.State != Ready || token == "" {
		return platformview.Session{}, errors.New("Platform Host is not ready")
	}
	session, err := platformview.Fetch(ctx, state.Endpoint.Origin, token)
	s.mu.Lock()
	current := s.gen == generation && s.platformToken == token && s.status.State == Ready
	s.mu.Unlock()
	if !current {
		return platformview.Session{}, errors.New("Platform Host changed; reopen the account page")
	}
	return session, err
}

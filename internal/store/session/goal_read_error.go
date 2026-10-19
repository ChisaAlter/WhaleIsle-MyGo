package session

import "whaleisle/internal/dsh"

// A background goal read owns its diagnostic, not the conversation's action
// status. Bind it to the exact selected follow so a new selection/Host cannot
// inherit it, including an A -> B -> A selection of the same session ID.
type goalReadErrorState struct {
	client     *dsh.Client
	sessionID  string
	generation uint64
	message    string
}

func (s *Store) goalReadErrorLocked() string {
	err := s.goalReadError
	if err.client != s.client || err.sessionID != s.selectedID || err.generation != s.followGeneration {
		return ""
	}
	return err.message
}

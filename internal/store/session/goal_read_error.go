package session

import (
	"whaleisle/internal/dsh"
	"whaleisle/internal/state"
)

// A background goal read owns its diagnostic, not the conversation's action
// status. Bind it to the exact selected follow so a new selection/Host cannot
// inherit it, including an A -> B -> A selection of the same session ID.
type goalReadErrorState struct {
	client     *dsh.Client
	sessionID  string
	generation uint64
	fault      *state.Fault
}

func (s *Store) goalReadErrorLocked() *state.Fault {
	err := s.goalReadError
	if err.client != s.client || err.sessionID != s.selectedID || err.generation != s.followGeneration {
		return nil
	}
	return err.fault
}

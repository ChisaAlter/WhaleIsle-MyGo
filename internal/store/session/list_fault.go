package session

import (
	"errors"
	"log"

	"whaleisle/internal/state"
)

// setListErrLocked records the list failure and its view form together, so
// every Snapshot and SelectionSummary hands out the same *state.Fault until the
// failure changes. The raw error stays here and in the boundary log only.
func (s *Store) setListErrLocked(err error) {
	if err == s.listErr {
		return
	}
	s.listErr = err
	s.listFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] list: %v", err)
	}
}

// setWorkspaceErrLocked is setListErrLocked for the workspace registry.
func (s *Store) setWorkspaceErrLocked(err error) {
	if err == s.workspaceErr {
		return
	}
	s.workspaceErr = err
	s.workspaceFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] workspace: %v", err)
	}
}

// errListSuperseded ends the retries of a list a newer refresh replaced.
var errListSuperseded = errors.New("session list superseded")

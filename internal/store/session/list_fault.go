package session

import (
	"errors"
	"log"

	"whaleisle/internal/state"
)

func faultEqual(a, b *state.Fault) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

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

func (s *Store) setHistoryErrLocked(prefix string, err error) {
	if err == nil && s.historyErr == nil && s.historyFault == nil {
		return
	}
	s.historyErr = err
	s.historyFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] %s%v", prefix, err)
	}
}

func (s *Store) setModelsErrLocked(err error) bool {
	next := state.FaultOf(err)
	changed := !faultEqual(s.modelsFault, next)
	s.modelsErr = err
	s.modelsFault = next
	if err != nil {
		log.Printf("[session] models: %v", err)
	}
	return changed
}

func (s *Store) setPermissionsErrLocked(err error) bool {
	next := state.FaultOf(err)
	changed := !faultEqual(s.permissionsFault, next)
	s.permissionsErr = err
	s.permissionsFault = next
	if err != nil {
		log.Printf("[session] permissions: %v", err)
	}
	return changed
}

func (s *Store) setAgentPresetsErrLocked(err error) bool {
	next := state.FaultOf(err)
	changed := !faultEqual(s.agentPresetsFault, next)
	s.agentPresetsErr = err
	s.agentPresetsFault = next
	if err != nil {
		log.Printf("[session] agentPresets: %v", err)
	}
	return changed
}

func (s *Store) setSkillsErrLocked(err error) {
	if err == s.skillsErr {
		return
	}
	s.skillsErr = err
	s.skillsFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] skills: %v", err)
	}
}

func (s *Store) setSearchErrLocked(err error) {
	if err == s.searchErr {
		return
	}
	s.searchErr = err
	s.searchFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] search: %v", err)
	}
}

func (s *Store) setHistoryPageErrLocked(err error) {
	if err == s.historyPageErr {
		return
	}
	s.historyPageErr = err
	s.historyPageFault = state.FaultOf(err)
	if err != nil {
		log.Printf("[session] page: %v", err)
	}
}

// errListSuperseded ends the retries of a list a newer refresh replaced.
var errListSuperseded = errors.New("session list superseded")

// faultText logs the raw failure at the store boundary and returns the safe,
// localized view sentence via state.FaultOf (which classifies transport/HTTP/RPC
// failures and preserves storelocale.AuthoredError copy verbatim).
func faultText(scope string, err error) string {
	if err == nil {
		return ""
	}
	log.Printf("[session] %s: %v", scope, err)
	return state.FaultOf(err).SentenceText()
}

// prefixedFaultText logs err and returns prefix + safe localized sentence.
// Used where existing UI/protocol contracts match a stable prefix such as
// "queue: ", "feedback: ", or "workspace follow: ".
func prefixedFaultText(scope, prefix string, err error) string {
	if err == nil {
		return ""
	}
	return prefix + faultText(scope, err)
}

// setLastActionFaultLocked logs err, sets s.lastAction to the classified
// localized sentence, and invalidates the view cache when the value changes.
func (s *Store) setLastActionFaultLocked(scope, prefix string, err error) {
	if err == nil {
		return
	}
	next := prefixedFaultText(scope, prefix, err)
	if s.lastAction != next {
		s.invalidateViewLocked()
	}
	s.lastAction = next
}

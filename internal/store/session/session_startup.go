package session

import "strings"

func isPluginOwnedSession(it Item) bool {
	if it.Presentation != nil && it.Presentation.Owner != "" && it.Presentation.Owner != "whaleisle" {
		return true
	}
	return strings.HasPrefix(it.SessionID, "session-whale-")
}

func (s *Store) hasOrdinarySessionsLocked() bool {
	for _, it := range s.items {
		if !isPluginOwnedSession(it) {
			return true
		}
	}
	return false
}

// initialWorkspaceSessionState belongs to an attached Host, not to a reconnecting
// events stream. Navigation or a completed initial decision never rearms it.
type initialWorkspaceSessionState struct {
	binding               uint64
	armed                 bool
	accepted              bool
	completed             bool
	eligible              bool
	pending               bool
	requestHost           uint64
	navigation, follow    uint64
	workspaceID, cwd      string
	defaultID, defaultCwd string
}

func (s *Store) reconcileInitialDefaultWorkspaceLocked() {
	initial := &s.initialWorkspaceSession
	if !initial.armed || !initial.completed || initial.pending || initial.accepted || !initial.eligible {
		return
	}
	if initial.defaultID == "" || initial.defaultCwd == "" {
		return
	}
	if !s.workspaceReady || s.workspaceReadyHost != s.eventsGeneration || s.workspaceErr != nil {
		return
	}
	if s.selectedID != "" || s.hasOrdinarySessionsLocked() || s.workspaceNavigation.view.Mode != "" {
		return
	}
	if s.workspaceNavigationToken != initial.navigation || s.followGeneration != initial.follow || s.cwd != initial.cwd {
		return
	}
	for _, row := range s.workspaceRowsLocked() {
		if hostPathKey(row.Path) == hostPathKey(s.cwd) {
			s.workspaceID = row.WorkspaceID
			initial.eligible = false
			return
		}
		if row.WorkspaceID != initial.defaultID && hostPathKey(row.Path) != hostPathKey(initial.defaultCwd) {
			initial.eligible = false
			return
		}
	}
	ws, ok := s.workspaceMemberships[initial.defaultID]
	if !ok || len(ws.SessionIDs) != 0 {
		return
	}
	s.workspaceID = initial.defaultID
	s.cwd = ws.Path
	initial.accepted = true
	initial.workspaceID = initial.defaultID
	initial.cwd = ws.Path
}

// InitialWorkspaceSession is the small startup receipt used by the UI owner.
// Ready means both Host baselines and the accepted default workspace are current.
type InitialWorkspaceSession struct {
	Binding         uint64
	HostGeneration  uint64
	NavigationToken uint64
	WorkspaceID     string
	Ready           bool
}

// InitialWorkspaceSession reads only startup authority; it does not build the
// chat snapshot or create/select a session.
func (s *Store) InitialWorkspaceSession() InitialWorkspaceSession {
	if s == nil {
		return InitialWorkspaceSession{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	initial := s.initialWorkspaceSession
	receipt := InitialWorkspaceSession{Binding: initial.binding, HostGeneration: s.eventsGeneration, NavigationToken: initial.navigation, WorkspaceID: initial.workspaceID}
	ws, registered := s.workspaceMemberships[initial.workspaceID]
	receipt.Ready = initial.armed && initial.eligible && initial.accepted && !initial.pending && s.client != nil && s.mux != nil &&
		s.workspaceReady && s.workspaceReadyHost == s.eventsGeneration && s.workspaceErr == nil &&
		s.listAuthoritativeHost == s.eventsGeneration && !s.listLoading && s.listErr == nil &&
		s.selectedID == "" && !s.hasOrdinarySessionsLocked() && s.workspaceNavigation.view.Mode == "" &&
		s.workspaceNavigationToken == initial.navigation && s.followGeneration == initial.follow &&
		registered && ws.Path == initial.cwd && len(ws.SessionIDs) == 0 && s.cwd == initial.cwd && s.workspaceID == initial.workspaceID
	return receipt
}

package session

// initialWorkspaceSessionState belongs to an attached Host, not to a reconnecting
// events stream. Navigation or a completed initial decision never rearms it.
type initialWorkspaceSessionState struct {
	binding            uint64
	armed              bool
	accepted           bool
	completed          bool
	eligible           bool
	pending            bool
	requestHost        uint64
	navigation, follow uint64
	workspaceID, cwd   string
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
		s.selectedID == "" && len(s.items) == 0 && s.workspaceNavigation.view.Mode == "" &&
		s.workspaceNavigationToken == initial.navigation && s.followGeneration == initial.follow &&
		registered && ws.Path == initial.cwd && len(ws.SessionIDs) == 0 && s.cwd == initial.cwd && s.workspaceID == initial.workspaceID
	return receipt
}

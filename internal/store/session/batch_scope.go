package session

// SessionBatchScope matches the browser's wide/group/archive/query key.
// Disclosure expansion and row ordering do not change the selection scope.
type SessionBatchScope struct {
	Wide                           bool
	GroupBy, ArchivedFilter, Query string
}

func NewSessionBatchScope(wide bool, groupBy, archivedFilter, query string) SessionBatchScope {
	if groupBy == "" {
		groupBy = "workspace"
	}
	if archivedFilter != "show" && archivedFilter != "only" {
		archivedFilter = "default"
	}
	return SessionBatchScope{Wide: wide, GroupBy: groupBy, ArchivedFilter: archivedFilter, Query: NormalizeSessionSearchQuery(query)}
}

// SetSessionBatchScope is called before rendering the browser's batch state.
// Choices clear immediately; a running operation keeps its frozen targets/token.
func (s *Store) SetSessionBatchScope(scope SessionBatchScope) {
	scope = NewSessionBatchScope(scope.Wide, scope.GroupBy, scope.ArchivedFilter, scope.Query)
	s.mu.Lock()
	s.ensureBatchLocked()
	if !s.batch.scopeSet {
		s.batch.scope, s.batch.scopeSet = scope, true
		s.mu.Unlock()
		return
	}
	if s.batch.scope == scope {
		s.mu.Unlock()
		return
	}
	s.batch.scope = scope
	s.batch.Selected = map[string]string{}
	if !s.batch.Busy {
		s.batchToken++
		s.batch.Token = s.batchToken
		s.batch.Confirming = false
		s.batch.Targets, s.batch.Results = nil, nil
		s.batch.Kind = ""
		s.batch.Attempted = 0
	}
	s.mu.Unlock()
	s.fire()
}

package session

// ProjectionKey names the store revision a view was projected from. Views with
// the same valid key carry identical Messages and WorkflowRuns: the store
// projects its cache once per revision and never edits it afterwards. A
// read-only projection derived only from those fields may therefore be reused
// across frames under the key. Views assembled by hand have the zero key.
//
// Keys compare by identity (==), while reflect.DeepEqual still treats two views
// with equal fields as equal: each revision owns a distinct token whose
// content is always zero.
type ProjectionKey struct {
	token *projectionToken
}

// WithoutProjectionKeys returns the view with its store revision keys cleared.
// A Snapshot copy may be edited freely, but it keeps the keys of the revision it
// was cut from; once its Messages, WorkflowRuns, Items or Workspaces differ from
// that revision the keys no longer describe it, and a memo that trusts them would
// serve the projection of the unedited view. Call it on a copy after editing it.
func (v ViewState) WithoutProjectionKeys() ViewState {
	v.projection, v.historyProjection, v.rosterProjection = ProjectionKey{}, ProjectionKey{}, ProjectionKey{}
	return v
}

// projectionToken has a non-zero size so every allocation has its own address.
type projectionToken struct{ _ byte }

func (k ProjectionKey) Valid() bool { return k.token != nil }

// ProjectionKey reports the store revision this view was projected from.
func (v ViewState) ProjectionKey() ProjectionKey { return v.projection }

// HistoryProjectionKey reports the message/workflow history revision this view
// was projected from. Unlike ProjectionKey, it remains stable across store
// mutations that do not alter Messages or WorkflowRuns.
func (v ViewState) HistoryProjectionKey() ProjectionKey {
	if v.historyProjection.Valid() {
		return v.historyProjection
	}
	return v.projection
}

// RosterProjectionKey reports the workspace/session roster revision this view
// was projected from. Unlike ProjectionKey, it remains stable across streaming
// message deltas that do not alter Items, Workspaces, WorkspaceLayout, or selection.
func (v ViewState) RosterProjectionKey() ProjectionKey {
	if v.rosterProjection.Valid() {
		return v.rosterProjection
	}
	return v.projection
}

// projectionKeyLocked names the revision being projected.
func (s *Store) projectionKeyLocked() ProjectionKey {
	if s.projectionToken == nil || s.projectionTokenRev != s.viewRev {
		s.projectionToken, s.projectionTokenRev = new(projectionToken), s.viewRev
	}
	return ProjectionKey{token: s.projectionToken}
}

func (s *Store) historyProjectionKeyLocked(changed bool) ProjectionKey {
	if s.historyProjectionToken == nil || changed {
		s.historyProjectionToken = new(projectionToken)
	}
	return ProjectionKey{token: s.historyProjectionToken}
}

func (s *Store) rosterProjectionKeyLocked(changed bool) ProjectionKey {
	if s.rosterProjectionToken == nil || changed {
		s.rosterProjectionToken = new(projectionToken)
	}
	return ProjectionKey{token: s.rosterProjectionToken}
}

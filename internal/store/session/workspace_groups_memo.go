package session

import "sync"

// workspaceGroupsMemo keeps recent groupings per store revision and item set.
// The sidebar asks for the same groups several times a frame (list, empty
// state, batch actions, drag markers) and the view is rebuilt on every hover,
// so regrouping every session each time dominated hover frames. Items are
// filtered copies of the view's own rows, so within one revision their ids
// determine them.
var workspaceGroupsMemo struct {
	sync.Mutex
	entries []workspaceGroupsEntry // most recently used first
}

type workspaceGroupsEntry struct {
	key         ProjectionKey
	ids         []string
	out         []WorkspaceGroup
	hoistPinned bool
}

const workspaceGroupsMemoSize = 4

// BuildWorkspaceGroups groups items by workspace account for the sidebar.
// Results for a store-projected view are shared across callers and frames;
// callers must not modify them.
func BuildWorkspaceGroups(snap ViewState, items []Item) []WorkspaceGroup {
	return memoWorkspaceGroups(snap, items, false)
}

// BuildSidebarWorkspaceGroups moves ordinary pinned rows into their shared
// leading section. Account grouping callers retain BuildWorkspaceGroups.
func BuildSidebarWorkspaceGroups(snap ViewState, items []Item) []WorkspaceGroup {
	return memoWorkspaceGroups(snap, items, true)
}

func memoWorkspaceGroups(snap ViewState, items []Item, hoistPinned bool) []WorkspaceGroup {
	key := snap.RosterProjectionKey()
	if !key.Valid() {
		return buildWorkspaceGroupsForSidebar(snap, items, hoistPinned)
	}
	workspaceGroupsMemo.Lock()
	for i, e := range workspaceGroupsMemo.entries {
		if e.key == key && e.hoistPinned == hoistPinned && sameItemIDs(e.ids, items) {
			copy(workspaceGroupsMemo.entries[1:i+1], workspaceGroupsMemo.entries[:i])
			workspaceGroupsMemo.entries[0] = e
			workspaceGroupsMemo.Unlock()
			return e.out
		}
	}
	workspaceGroupsMemo.Unlock()
	out := buildWorkspaceGroupsForSidebar(snap, items, hoistPinned)
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.SessionID
	}
	workspaceGroupsMemo.Lock()
	entries := append([]workspaceGroupsEntry{{key: key, ids: ids, out: out, hoistPinned: hoistPinned}}, workspaceGroupsMemo.entries...)
	if len(entries) > workspaceGroupsMemoSize {
		entries = entries[:workspaceGroupsMemoSize]
	}
	workspaceGroupsMemo.entries = entries
	workspaceGroupsMemo.Unlock()
	return out
}

func sameItemIDs(ids []string, items []Item) bool {
	if len(ids) != len(items) {
		return false
	}
	for i, it := range items {
		if ids[i] != it.SessionID {
			return false
		}
	}
	return true
}

var workspaceOwnersMemo struct {
	sync.Mutex
	key    ProjectionKey
	ids    []string
	owners map[string]string
}

// WorkspaceSessionOwnersForView memoizes WorkspaceSessionOwners for a projected view.
func WorkspaceSessionOwnersForView(snap ViewState, items []Item) map[string]string {
	key := snap.RosterProjectionKey()
	if !key.Valid() {
		return WorkspaceSessionOwners(snap.Workspaces, items)
	}
	workspaceOwnersMemo.Lock()
	if workspaceOwnersMemo.key == key && sameItemIDs(workspaceOwnersMemo.ids, items) {
		out := workspaceOwnersMemo.owners
		workspaceOwnersMemo.Unlock()
		return out
	}
	workspaceOwnersMemo.Unlock()
	out := WorkspaceSessionOwners(snap.Workspaces, items)
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.SessionID
	}
	workspaceOwnersMemo.Lock()
	workspaceOwnersMemo.key = key
	workspaceOwnersMemo.ids = ids
	workspaceOwnersMemo.owners = out
	workspaceOwnersMemo.Unlock()
	return out
}

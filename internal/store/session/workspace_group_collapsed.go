package session

import "sync"

var workspaceDefaultOpenMemo struct {
	sync.Mutex
	key  ProjectionKey
	open map[string]bool
}

// WorkspaceGroupCollapsed resolves the browser's default expansion before an
// explicit choice exists. Flat rows have no group disclosure. A stored false
// is an explicit open choice and must not be confused with an absent map key.
func WorkspaceGroupCollapsed(snap ViewState, key string) bool {
	if snap.WorkspaceLayout.GroupBy == "flat" || key == "::flat" {
		return false
	}
	if collapsed, explicit := snap.WorkspaceLayout.Collapsed[key]; explicit {
		return collapsed
	}
	if key == PinnedWorkspaceGroup {
		return false
	}
	projKey := snap.RosterProjectionKey()
	if projKey.Valid() {
		workspaceDefaultOpenMemo.Lock()
		if workspaceDefaultOpenMemo.key == projKey && workspaceDefaultOpenMemo.open != nil {
			open := workspaceDefaultOpenMemo.open[key]
			workspaceDefaultOpenMemo.Unlock()
			return !open
		}
		workspaceDefaultOpenMemo.Unlock()
	}
	open := computeDefaultOpenWorkspaceGroups(snap)
	if projKey.Valid() {
		workspaceDefaultOpenMemo.Lock()
		workspaceDefaultOpenMemo.key = projKey
		workspaceDefaultOpenMemo.open = open
		workspaceDefaultOpenMemo.Unlock()
	}
	return !open[key]
}

func computeDefaultOpenWorkspaceGroups(snap ViewState) map[string]bool {
	open := map[string]bool{}
	parents := map[string]string{}
	if snap.WorkspaceLayout.GroupBy == "workspace-tree" {
		parents = registeredWorkspaceParents(snap.Workspaces)
		for _, parent := range parents {
			if parent != "" {
				open[parent] = true
			}
		}
	}
	hasPresentation := make(map[string]bool, len(snap.Items))
	selectedNoDirectory := false
	for _, item := range snap.Items {
		if item.Presentation != nil {
			hasPresentation[item.SessionID] = true
		}
		if snap.SelectedID != "" && item.SessionID == snap.SelectedID && snap.IsNoDirectory(item) {
			selectedNoDirectory = true
		}
	}
	current := ""
	for _, workspace := range snap.Workspaces {
		for _, id := range workspace.SessionIDs {
			if id == snap.SelectedID && id != "" && current == "" {
				current = workspace.WorkspaceID
			}
			if hasPresentation[id] {
				open[workspace.WorkspaceID] = true
			}
		}
	}
	if current == "" && selectedNoDirectory {
		current = "::scratch"
	}
	for visited := map[string]bool{}; current != "" && !visited[current]; current = parents[current] {
		open[current] = true
		visited[current] = true
	}
	return open
}

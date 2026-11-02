package sidebar

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type sidebarDragScope struct {
	Generation                              uint64
	GroupBy, OrderBy, ArchiveFilter, Search string
	Batch                                   bool
}

type workspaceDropMarker struct {
	Source workspaceDrag
	Scope  sidebarDragScope
	Target string
	After  bool
}

type sessionDropMarker struct {
	Source       sessionDrag
	Scope        sidebarDragScope
	Target       string
	After, Local bool
}

// sidebarDragState belongs to one sidebar occurrence, never the selected cwd.
// Markers survive leaving the list, but every completion revalidates ownership.
type sidebarDragState struct {
	HostGeneration uint64
	Workspace      *workspaceDropMarker
	Session        *sessionDropMarker
}

func currentSidebarDragScope(m Model) sidebarDragScope {
	query := ""
	if m.SearchQuery != nil {
		query = strings.TrimSpace(*m.SearchQuery)
	}
	layout := m.Snapshot.WorkspaceLayout
	return sidebarDragScope{layout.HostGeneration, layout.GroupBy, layout.OrderBy, browseArchiveFilter(m), query, m.Batch.Enabled}
}

func workspaceDragModel(list ui.Element, m Model) Model {
	if m.sidebarDrag == nil {
		m.sidebarDrag = ui.Local(list, "sidebar-drag-state", func() sidebarDragState { return sidebarDragState{} })
	}
	if m.sidebarDrag.HostGeneration != m.Snapshot.WorkspaceLayout.HostGeneration {
		*m.sidebarDrag = sidebarDragState{HostGeneration: m.Snapshot.WorkspaceLayout.HostGeneration}
	}
	validateSidebarDragMarkers(m)
	return m
}

func validateSidebarDragMarkers(m Model) {
	state, scope := m.sidebarDrag, currentSidebarDragScope(m)
	if state == nil {
		return
	}
	if m.Snapshot.WorkspaceLayout.Reordering || m.Batch.Enabled {
		state.Workspace, state.Session = nil, nil
		return
	}
	if state.Workspace == nil && state.Session == nil {
		return
	}
	groups := currentDragGroups(m)
	if marker := state.Workspace; marker != nil {
		_, valid := session.WorkspaceDropAnchor(groups, marker.Source.ID, marker.Target, marker.After)
		if marker.Scope != scope || marker.Source.Scope != scope || marker.Source.HostGeneration != scope.Generation || !valid {
			state.Workspace = nil
		}
	}
	if marker := state.Session; marker != nil {
		valid := false
		if marker.Scope == scope && marker.Source.Scope == scope && marker.Source.Generation == scope.Generation {
			for _, group := range groups {
				if group.Key != marker.Source.Workspace || session.WorkspaceGroupCollapsed(m.Snapshot, group.Key) {
					continue
				}
				rows := visibleWorkspaceSessionRows(m, group).Rows
				_, valid = sessionDragAnchor(group.Workspace.SessionIDs, rows, marker.Source.ID, marker.Target, marker.After)
				for _, row := range rows {
					if row.Item.SessionID == marker.Source.ID && (row.Item.Pinned != marker.Source.Pinned || isArchived(m.Snapshot, row.Item.SessionID)) {
						valid = false
					}
				}
				break
			}
		}
		if !valid {
			state.Session = nil
		}
	}
}

func currentDragGroups(m Model) []session.WorkspaceGroup {
	items := sidebarItemsForArchiveFilter(m.Snapshot, m.SearchQuery, browseArchiveFilter(m))
	return visibleWorkspaceGroups(m, items)
}

func beginSidebarDrag(e ui.Element, m Model, workspace bool) {
	if e.Pressed() && !e.Dragging() && m.sidebarDrag != nil {
		if workspace {
			m.sidebarDrag.Workspace = nil
		} else {
			m.sidebarDrag.Session = nil
		}
	}
}

func workspaceDragSource(e ui.Element, m Model, id string) {
	e.Drag(workspaceDrag{HostGeneration: m.Snapshot.WorkspaceLayout.HostGeneration, ID: id, Scope: currentSidebarDragScope(m)})
	beginSidebarDrag(e, m, true)
	end, completed := ui.DragEnd[workspaceDrag](e)
	if !completed || m.sidebarDrag == nil {
		return
	}
	marker := m.sidebarDrag.Workspace
	m.sidebarDrag.Workspace = nil
	if end.Canceled || end.Dropped || marker == nil || marker.Source != end.Value ||
		marker.Scope != currentSidebarDragScope(m) || m.Snapshot.WorkspaceLayout.Reordering || m.OnDropWorkspaceBefore == nil {
		return
	}
	before, ok := session.WorkspaceDropAnchor(currentDragGroups(m), end.Value.ID, marker.Target, marker.After)
	if ok {
		m.OnDropWorkspaceBefore(end.Value.HostGeneration, end.Value.ID, before)
	}
}

func completeSessionDrag(e ui.Element, m Model) {
	end, completed := ui.DragEnd[sessionDrag](e)
	if !completed || m.sidebarDrag == nil {
		return
	}
	marker := m.sidebarDrag.Session
	m.sidebarDrag.Session = nil
	if end.Canceled || end.Dropped || marker == nil || marker.Source != end.Value ||
		marker.Scope != currentSidebarDragScope(m) || m.Snapshot.WorkspaceLayout.Reordering || m.Batch.Enabled {
		return
	}
	for _, group := range currentDragGroups(m) {
		if group.Key != end.Value.Workspace || session.WorkspaceGroupCollapsed(m.Snapshot, group.Key) {
			continue
		}
		visible := visibleWorkspaceSessionRows(m, group)
		for _, row := range visible.Rows {
			if row.Item.SessionID == end.Value.ID && row.Item.Pinned != end.Value.Pinned {
				return
			}
		}
		before, ok := sessionDragAnchor(group.Workspace.SessionIDs, visible.Rows, end.Value.ID, marker.Target, marker.After)
		if !ok {
			return
		}
		if marker.Local {
			if m.OnMoveLocalSessionBefore != nil {
				m.OnMoveLocalSessionBefore(end.Value.Generation, group.Key, end.Value.ID, before)
			}
		} else if group.Kind == "workspace" && m.OnMoveWorkspaceSessionBefore != nil {
			m.OnMoveWorkspaceSessionBefore(group.Key, end.Value.ID, before)
		}
		return
	}
}

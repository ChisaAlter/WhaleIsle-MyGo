package sidebar

import (
	"slices"
	"strings"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/store/session"
)

// Owner membership is a display projection. Never replace the Session's cwd or
// register a synthetic directory with the Host just to make its row visible.
func workspaceOwnerNavigationModel(m Model) Model {
	if len(m.WorkspaceNavigation) == 0 {
		return m
	}
	byID := make(map[string]int, len(m.Snapshot.Items))
	for i, item := range m.Snapshot.Items {
		byID[item.SessionID] = i
	}
	m.ownerNavigation = map[string]pluginui.WorkspaceNavigation{}
	for _, entry := range m.WorkspaceNavigation {
		i, found := byID[entry.SessionID]
		if !found || entry.Owner == "" || entry.Owner == "project" || entry.Owner == "whaleisle" {
			continue
		}
		p := m.Snapshot.Items[i].Presentation
		if p != nil && p.Owner == entry.Owner && p.WorkingDirectory == entry.WorkingDirectory {
			m.ownerNavigation[entry.SessionID] = entry
		}
	}
	if len(m.ownerNavigation) == 0 {
		return m
	}
	snap := m.Snapshot.WithoutProjectionKeys()
	snap.Items = slices.Clone(snap.Items)
	snap.Workspaces = slices.Clone(snap.Workspaces)
	snap.ArchivedSessionIDs = slices.Clone(snap.ArchivedSessionIDs)
	m.navigationOnly = map[string]bool{}
	for i := range snap.Workspaces {
		snap.Workspaces[i].SessionIDs = slices.DeleteFunc(slices.Clone(snap.Workspaces[i].SessionIDs), func(id string) bool {
			_, owned := m.ownerNavigation[id]
			return owned
		})
	}
	// Keep the source registration order, including its prepend membership rule.
	for _, entry := range m.WorkspaceNavigation {
		if _, valid := m.ownerNavigation[entry.SessionID]; !valid {
			continue
		}
		item := &snap.Items[byID[entry.SessionID]]
		item.Title, item.Blank = entry.Title, false
		if item.Title == "" {
			item.Title = m.Snapshot.Items[byID[entry.SessionID]].Presentation.Title
		}
		item.Presentation = nil // only this read-only sidebar copy is ordinary.
		snap.ArchivedSessionIDs = slices.DeleteFunc(snap.ArchivedSessionIDs, func(id string) bool { return id == entry.SessionID })
		if entry.Archived {
			snap.ArchivedSessionIDs = append(snap.ArchivedSessionIDs, entry.SessionID)
		}
		directory := -1
		for i := range snap.Workspaces {
			if snap.Workspaces[i].Path == entry.WorkingDirectory {
				directory = i
				break
			}
		}
		if directory < 0 {
			path := strings.TrimRight(strings.ReplaceAll(entry.WorkingDirectory, `\`, "/"), "/")
			title := path[strings.LastIndex(path, "/")+1:]
			if title == "" {
				title = entry.WorkingDirectory
			}
			id := "navigation:" + entry.WorkingDirectory
			snap.Workspaces = append(snap.Workspaces, session.WorkspaceRow{WorkspaceID: id, Path: entry.WorkingDirectory, Title: title})
			m.navigationOnly[id] = true
			directory = len(snap.Workspaces) - 1
		}
		snap.Workspaces[directory].SessionIDs = append([]string{entry.SessionID}, snap.Workspaces[directory].SessionIDs...)
	}
	m.Snapshot = snap
	return m
}

// The owner supplies every mutating action. Generic Session pin/fork/delete,
// batch operations and drag mutations are not implied by a navigation entry.
func workspaceOwnerRowModel(m Model, id string) Model {
	entry, owned := m.ownerNavigation[id]
	if !owned {
		return m
	}
	m.ownerRow = true
	run := func(action string) {
		if m.OnWorkspaceSessionAction != nil {
			m.OnWorkspaceSessionAction(id, entry.Owner, action)
		}
	}
	m.OnSelect = func(string) { run("open") }
	m.OnBeginRename, m.OnArchive, m.OnUnarchive = nil, nil, nil
	if m.OnWorkspaceSessionAction != nil {
		if entry.CanRename {
			m.OnBeginRename = func(string, string) { run("rename") }
		}
		if entry.CanArchive {
			m.OnArchive = func(string) { run("archive") }
		}
		if entry.CanRestore {
			m.OnUnarchive = func(string) { run("restore") }
		}
	}
	m.OnPin, m.OnUnpin, m.OnFork, m.OnDelete = nil, nil, nil, nil
	m.OnOpenSessionPlugins, m.OnCloseSessionPlugins = nil, nil
	m.sessionDrag, m.OnMoveSessionBefore = nil, nil
	return m
}

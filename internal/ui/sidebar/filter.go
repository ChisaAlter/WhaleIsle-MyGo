package sidebar

import (
	"strings"
	"sync"

	"whaleisle/internal/store/session"
)

var sidebarFilterMemo struct {
	sync.Mutex
	key    session.ProjectionKey
	query  string
	filter string
	items  []session.Item
}

func sidebarItems(snap session.ViewState, queryPtr *string) []session.Item {
	return sidebarItemsForArchiveFilter(snap, queryPtr, "default")
}

func sidebarItemsForArchiveFilter(snap session.ViewState, queryPtr *string, filter string) []session.Item {
	q := ""
	if queryPtr != nil {
		q = strings.TrimSpace(*queryPtr)
	}
	projKey := snap.RosterProjectionKey()
	if q != "" {
		projKey = snap.ProjectionKey()
	}
	if projKey.Valid() {
		sidebarFilterMemo.Lock()
		if sidebarFilterMemo.key == projKey && sidebarFilterMemo.query == q && sidebarFilterMemo.filter == filter {
			cached := sidebarFilterMemo.items
			sidebarFilterMemo.Unlock()
			return cached
		}
		sidebarFilterMemo.Unlock()
	}
	archived := make(map[string]bool, len(snap.ArchivedSessionIDs))
	for _, id := range snap.ArchivedSessionIDs {
		archived[id] = true
	}
	members := make(map[string]bool)
	for _, workspace := range snap.Workspaces {
		for _, id := range workspace.SessionIDs {
			members[id] = true
		}
	}
	items := make([]session.Item, 0, len(snap.Items))
	for _, it := range snap.Items {
		if it.SessionID == "" || it.Origin == "subagent" || it.Blank && it.SessionID != snap.SelectedID {
			continue
		}
		if it.Presentation != nil && it.Presentation.Owner != "" && it.Presentation.Owner != "whaleisle" {
			continue
		}
		if !members[it.SessionID] && !snap.IsNoDirectory(it) {
			continue
		}
		if searchArchiveVisible(archived[it.SessionID], filter) {
			items = append(items, it)
		}
	}
	if q != "" {
		rpcReady := !snap.SearchLoading &&
			!snap.SearchUseLocal &&
			snap.SearchErr == "" &&
			snap.SearchRPCQuery == q
		if rpcReady {
			items = filterItemsByHitIDs(items, snap.SearchHitIDs)
		} else {
			items = filterItems(items, q)
		}
	}
	if projKey.Valid() {
		sidebarFilterMemo.Lock()
		sidebarFilterMemo.key = projKey
		sidebarFilterMemo.query = q
		sidebarFilterMemo.filter = filter
		sidebarFilterMemo.items = items
		sidebarFilterMemo.Unlock()
	}
	return items
}

func isArchived(snap session.ViewState, id string) bool {
	for _, archivedID := range snap.ArchivedSessionIDs {
		if id == archivedID {
			return true
		}
	}
	return false
}

var sidebarArchivedMemo struct {
	sync.Mutex
	key              session.ProjectionKey
	filter           string
	hideArchivedList bool
	rows             []session.Item
	valid            bool
}

// Archived ids with missing summaries stay manageable; summaries outside a
// registered workspace or the scratch directory do not appear in the sidebar.
func archivedSectionItems(m Model) []session.Item {
	if len(m.Snapshot.ArchivedSessionIDs) == 0 {
		return nil
	}
	filter := browseArchiveFilter(m)
	if filter == "default" && m.HideArchivedList {
		return nil
	}
	projKey := m.Snapshot.RosterProjectionKey()
	if projKey.Valid() {
		sidebarArchivedMemo.Lock()
		if sidebarArchivedMemo.valid && sidebarArchivedMemo.key == projKey &&
			sidebarArchivedMemo.filter == filter && sidebarArchivedMemo.hideArchivedList == m.HideArchivedList {
			rows := sidebarArchivedMemo.rows
			sidebarArchivedMemo.Unlock()
			return rows
		}
		sidebarArchivedMemo.Unlock()
	}
	members := map[string]bool{}
	for _, workspace := range m.Snapshot.Workspaces {
		for _, id := range workspace.SessionIDs {
			members[id] = true
		}
	}
	byID := make(map[string]session.Item, len(m.Snapshot.Items))
	for _, item := range m.Snapshot.Items {
		byID[item.SessionID] = item
	}
	var rows []session.Item
	for _, id := range m.Snapshot.ArchivedSessionIDs {
		item, found := byID[id]
		if !found {
			rows = append(rows, session.Item{SessionID: id, Title: workspaceText("缺失会话", "Missing session")})
			continue
		}
		if item.Presentation != nil && item.Presentation.Owner != "" && item.Presentation.Owner != "whaleisle" {
			continue
		}
		if filter != "default" || item.Origin == "subagent" || !members[id] && !m.Snapshot.IsNoDirectory(item) {
			continue
		}
		rows = append(rows, item)
	}
	if projKey.Valid() {
		sidebarArchivedMemo.Lock()
		sidebarArchivedMemo.key = projKey
		sidebarArchivedMemo.filter = filter
		sidebarArchivedMemo.hideArchivedList = m.HideArchivedList
		sidebarArchivedMemo.rows = rows
		sidebarArchivedMemo.valid = true
		sidebarArchivedMemo.Unlock()
	}
	return rows
}

func filterItemsByHitIDs(items []session.Item, hits map[string]bool) []session.Item {
	if len(hits) == 0 {
		return nil
	}
	byID := make(map[string]session.Item, len(items))
	children := make(map[string][]string)
	for _, it := range items {
		byID[it.SessionID] = it
		if it.ParentID != "" {
			children[it.ParentID] = append(children[it.ParentID], it.SessionID)
		}
	}
	visible := map[string]bool{}
	for id, ok := range hits {
		if !ok {
			continue
		}
		it, ok := byID[id]
		if !ok {
			continue
		}
		visible[it.SessionID] = true
		for pid := it.ParentID; pid != ""; {
			visible[pid] = true
			pid = byID[pid].ParentID
		}
		var markDesc func(id string)
		markDesc = func(id string) {
			for _, cid := range children[id] {
				visible[cid] = true
				markDesc(cid)
			}
		}
		markDesc(it.SessionID)
	}
	out := make([]session.Item, 0, len(visible))
	for _, it := range items {
		if visible[it.SessionID] {
			out = append(out, it)
		}
	}
	return out
}

func filterItems(items []session.Item, query string) []session.Item {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return items
	}
	byID := make(map[string]session.Item, len(items))
	children := make(map[string][]string)
	for _, it := range items {
		byID[it.SessionID] = it
		if it.ParentID != "" {
			children[it.ParentID] = append(children[it.ParentID], it.SessionID)
		}
	}
	match := func(it session.Item) bool {
		return strings.Contains(strings.ToLower(it.Title), q) ||
			strings.Contains(strings.ToLower(it.SessionID), q)
	}
	visible := map[string]bool{}
	for _, it := range items {
		if !match(it) {
			continue
		}
		visible[it.SessionID] = true
		for pid := it.ParentID; pid != ""; {
			visible[pid] = true
			pid = byID[pid].ParentID
		}
		var markDesc func(id string)
		markDesc = func(id string) {
			for _, cid := range children[id] {
				visible[cid] = true
				markDesc(cid)
			}
		}
		markDesc(it.SessionID)
	}
	out := make([]session.Item, 0, len(visible))
	for _, it := range items {
		if visible[it.SessionID] {
			out = append(out, it)
		}
	}
	return out
}

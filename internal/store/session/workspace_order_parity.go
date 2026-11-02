package session

import (
	"slices"
	"sort"
	"strings"
)

// activeWorkspaceSessionOrdersLocked mirrors the browser's activeSessionOrders:
// all accounts are frozen, independently of the currently selected grouping,
// collapsed sections, search, or archive visibility. Membership stays on Host.
func (s *Store) activeWorkspaceSessionOrdersLocked() map[string][]string {
	snap := ViewState{
		Items: s.items, SelectedID: s.selectedID,
		Workspaces: s.workspaceRowsLocked(), WorkspaceLayout: s.workspaceLayoutLocked(),
		ScratchCwd: s.scratchCwd, WorkspaceMembers: s.workspaceMembersLocked(),
		PinnedSessionIDs: s.pinnedSessionIDs, ArchivedSessionIDs: s.archivedSessionIDs,
	}
	orders := map[string][]string{}
	accounted := map[string]bool{}
	index := newWorkspaceOrderIndex(snap)
	for _, workspace := range snap.Workspaces {
		orders[workspace.WorkspaceID] = index.order(snap, workspace.SessionIDs, workspace.WorkspaceID)
		for _, id := range workspace.SessionIDs {
			accounted[id] = true
		}
	}
	var ungrouped, flat []string
	for _, item := range snap.Items {
		if !accounted[item.SessionID] {
			ungrouped = append(ungrouped, item.SessionID)
		}
		if item.SessionID == "" || item.Origin == "subagent" || item.Blank && item.SessionID != snap.SelectedID || item.Presentation != nil && item.Presentation.Owner != "" && item.Presentation.Owner != "whaleisle" {
			continue
		}
		if accounted[item.SessionID] || snap.IsNoDirectory(item) {
			flat = append(flat, item.SessionID)
		}
	}
	orders["::scratch"] = index.order(snap, ungrouped, "::scratch")
	orders["::flat"] = index.order(snap, flat, "::flat")
	return orders
}

func (s *Store) setWorkspaceOrderingLocked(mode string) bool {
	if s.workspaceLayoutLocked().OrderBy == mode {
		return false
	}
	orders := map[string][]string{}
	if mode == "manual" {
		orders = s.activeWorkspaceSessionOrdersLocked()
	}
	s.workspaceLayout.view.OrderBy = mode
	s.workspaceLayout.view.SessionOrders = orders
	s.workspacePreferencesChangedLocked("ordering", "")
	return true
}

// insertWorkspaceSessionLocalLocked applies an insert-before gesture to one
// complete local account, preserving other accounts and hidden member slots.
func (s *Store) insertWorkspaceSessionLocalLocked(generation uint64, key, id, before string) bool {
	key, id, before = strings.TrimSpace(key), strings.TrimSpace(id), strings.TrimSpace(before)
	if generation != s.eventsGeneration || !s.workspaceReady || s.client == nil || id == "" || id == before {
		return false
	}
	orders := s.activeWorkspaceSessionOrdersLocked()
	order, exists := orders[key]
	at := slices.Index(order, id)
	if !exists || at < 0 || before != "" && !slices.Contains(order, before) {
		return false
	}
	byID := map[string]Item{}
	for _, item := range s.items {
		byID[item.SessionID] = item
	}
	visible := func(item Item) bool {
		if item.Origin == "subagent" || item.Blank && item.SessionID != s.selectedID || !workspaceOrderItemVisible(item, s.workspaceLayoutLocked().ArchivedFilter, s.archivedSessionIDs) {
			return false
		}
		return key != "::scratch" || s.scratchCwd != "" && filepathClean(item.Cwd) == filepathClean(s.scratchCwd)
	}
	source, known := byID[id]
	if !known || source.Blank || slices.Contains(s.archivedSessionIDs, id) || !visible(source) {
		return false
	}
	// The UI validates actual drop targets and the pin partition. Its insert-
	// before anchor may instead be the next hidden, archived, or pinned slot.
	// Preserve that complete-account anchor rather than re-filtering it here.
	if target, known := byID[before]; before != "" && known && target.Blank {
		return false
	}
	next := slices.Delete(slices.Clone(order), at, at+1)
	insertAt := len(next)
	if before != "" {
		insertAt = slices.Index(next, before)
	}
	next = slices.Insert(next, insertAt, id)
	if slices.Equal(order, next) {
		return false
	}
	orders[key] = next
	s.workspaceLayout.view.OrderBy = "manual"
	s.workspaceLayout.view.SessionOrders = orders
	s.workspaceLayout.view.Error = ""
	s.lastAction = "session reordered"
	s.workspacePreferencesChangedLocked("ordering", "")
	return true
}

func workspaceOrderItemVisible(item Item, filter string, archived []string) bool {
	if item.Presentation != nil && item.Presentation.Owner != "" && item.Presentation.Owner != "whaleisle" {
		return false
	}
	isArchived := slices.Contains(archived, item.SessionID)
	return filter == "show" || filter == "only" && isArchived || filter != "only" && !isArchived
}

// InsertSessionBeforeForHost fences drag gestures captured by an older Host.
func (s *Store) InsertSessionBeforeForHost(generation uint64, workspaceID, sessionID, beforeSessionID string) {
	s.mu.Lock()
	changed := s.insertWorkspaceSessionLocalLocked(generation, workspaceID, sessionID, beforeSessionID)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

// workspaceOrderIndex holds the per-view lookups of workspaceAccountSessionOrder.
// Callers order every account of one view, so build it once per view: rebuilding
// it from every session for each workspace dominated the sidebar's frame cost.
type workspaceOrderIndex struct {
	byID     map[string]Item
	archived map[string]bool
}

func newWorkspaceOrderIndex(snap ViewState) workspaceOrderIndex {
	idx := workspaceOrderIndex{byID: make(map[string]Item, len(snap.Items)), archived: make(map[string]bool, len(snap.ArchivedSessionIDs))}
	for _, item := range snap.Items {
		idx.byID[item.SessionID] = item
	}
	for _, id := range snap.ArchivedSessionIDs {
		idx.archived[id] = true
	}
	return idx
}

func workspaceAccountSessionOrder(snap ViewState, memberIDs []string, key string) []string {
	return newWorkspaceOrderIndex(snap).order(snap, memberIDs, key)
}

// order needs an index built from the same snap; it only reads the index.
func (idx workspaceOrderIndex) order(snap ViewState, memberIDs []string, key string) []string {
	byID, archived := idx.byID, idx.archived
	members, included := map[string]bool{}, map[string]bool{}
	for _, id := range memberIDs {
		members[id] = true
	}
	// Sort extracted keys: fetching both Items per comparison dominated. byID is
	// keyed by SessionID, so the id is the item's SessionID.
	type recent struct {
		id        string
		updatedAt float64
	}
	var ranked []recent
	for id := range members {
		if item, known := byID[id]; known {
			ranked = append(ranked, recent{id, item.UpdatedAt})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].updatedAt != ranked[j].updatedAt {
			return ranked[i].updatedAt > ranked[j].updatedAt
		}
		return ranked[i].id < ranked[j].id
	})
	var recency []string
	for _, r := range ranked {
		recency = append(recency, r.id)
	}
	result := recency
	if snap.WorkspaceLayout.OrderBy == "manual" {
		var saved, pins, ordinary, archives []string
		for _, id := range snap.WorkspaceLayout.SessionOrders[key] {
			if members[id] && !included[id] {
				saved = append(saved, id)
				included[id] = true
			}
		}
		for _, id := range snap.PinnedSessionIDs {
			if _, known := byID[id]; members[id] && known && !included[id] && !archived[id] {
				pins = append(pins, id)
				included[id] = true
			}
		}
		for _, id := range recency {
			if included[id] {
				continue
			}
			if archived[id] {
				archives = append(archives, id)
			} else {
				ordinary = append(ordinary, id)
			}
		}
		result = make([]string, 0, len(pins)+len(saved)+len(ordinary)+len(archives))
		result = append(result, pins...)
		result = append(result, saved...)
		result = append(result, ordinary...)
		result = append(result, archives...)
		pending := map[string]bool{}
		for _, id := range ordinary {
			pending[id] = true
		}
		index := func(id string) int {
			for i, candidate := range result {
				if candidate == id {
					return i
				}
			}
			return -1
		}
		var placeFork func(string)
		placeFork = func(id string) {
			if !pending[id] {
				return
			}
			delete(pending, id)
			parent := byID[id].ParentID
			if parent == "" || parent == id || index(parent) < 0 {
				return
			}
			placeFork(parent)
			at := index(id)
			result = append(result[:at], result[at+1:]...)
			at = index(parent)
			result = append(result, "")
			copy(result[at+1:], result[at:])
			result[at] = id
		}
		for i := len(ordinary) - 1; i >= 0; i-- {
			placeFork(ordinary[i])
		}
	}
	if current, known := byID[snap.SelectedID]; known && current.Blank && members[current.SessionID] {
		out := []string{current.SessionID}
		for _, id := range result {
			if id != current.SessionID {
				out = append(out, id)
			}
		}
		result = out
	}
	return result
}

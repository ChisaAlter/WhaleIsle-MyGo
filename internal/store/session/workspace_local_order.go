package session

import "slices"

func cloneSessionOrders(in map[string][]string) map[string][]string {
	out := map[string][]string{}
	for k, v := range in {
		out[k] = slices.Clone(v)
	}
	return out
}

// Local display orders are separate from real Host workspace membership.
// Reconciliation uses the full roster, including rows hidden by UI filters.
func localSessionOrder(snap ViewState, key string) []string {
	owners := WorkspaceSessionOwners(snap.Workspaces, snap.Items)
	members := map[string]bool{}
	var base []string
	for _, it := range snap.Items {
		include := key == "::flat" || (owners[it.SessionID] == "" && ((key == "::scratch" && snap.IsNoDirectory(it)) || (key == "::other" && !snap.IsNoDirectory(it))))
		if include && it.SessionID != "" && !members[it.SessionID] {
			members[it.SessionID] = true
			base = append(base, it.SessionID)
		}
	}
	var out []string
	for _, id := range snap.WorkspaceLayout.SessionOrders[key] {
		if members[id] && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	for _, id := range base {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
func (s *Store) InsertLocalSessionBefore(generation uint64, key, id, before string) {
	s.mu.Lock()
	if generation != s.eventsGeneration || !s.workspaceReady || s.workspaceLayout.view.OrderBy == "updated" || (key != "::flat" && key != "::scratch" && key != "::other") {
		s.mu.Unlock()
		return
	}
	snap := ViewState{Items: s.items, Workspaces: s.workspaceRowsLocked(), WorkspaceLayout: s.workspaceLayoutLocked(), ScratchCwd: s.scratchCwd}
	order := localSessionOrder(snap, key)
	at := slices.Index(order, id)
	if at < 0 || id == before || (before != "" && !slices.Contains(order, before)) {
		s.mu.Unlock()
		return
	}
	for _, it := range s.items {
		if it.SessionID == id && it.Blank {
			s.mu.Unlock()
			return
		}
	}
	order = slices.Delete(order, at, at+1)
	at = len(order)
	if before != "" {
		at = slices.Index(order, before)
	}
	order = slices.Insert(order, at, id)
	if s.workspaceLayout.view.SessionOrders == nil {
		s.workspaceLayout.view.SessionOrders = map[string][]string{}
	}
	s.workspaceLayout.view.SessionOrders[key] = order
	s.workspacePreferencesChangedLocked()
	s.mu.Unlock()
	s.fire()
}

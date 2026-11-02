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
	if key != "::flat" && key != "::scratch" {
		return
	}
	s.InsertSessionBeforeForHost(generation, key, id, before)
}

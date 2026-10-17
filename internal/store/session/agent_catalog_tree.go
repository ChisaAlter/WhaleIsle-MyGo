package session

import (
	"context"
	"encoding/json"
	"time"
)

type AgentCatalogTreeRow struct {
	Row            AgentNavigationRow
	Path           []AgentSessionAddress
	Depth          int
	Expanded, Busy bool
	Error          string
}
type agentCatalogBranch struct {
	view           AgentNavigationView
	expanded, busy bool
	error          string
	cancel         context.CancelFunc
	request        uint64
}

func agentCatalogKey(base AgentSessionAddress, path []AgentSessionAddress) string {
	b, _ := json.Marshal(struct {
		Base AgentSessionAddress
		Path []AgentSessionAddress
	}{base, path})
	return string(b)
}

// Resolve only a current, explicitly expanded chain. Every edge must still be
// present in the authoritative parent catalog; a cached child ID is not enough.
func (s *Store) agentCatalogPathLocked(path []AgentSessionAddress) ([]AgentNavigationView, bool) {
	if len(path) == 0 || len(path) > 64 {
		return nil, false
	}
	n := &s.agentNavigation
	views := []AgentNavigationView{n.view}
	seen := map[string]bool{n.view.SessionID: true}
	for i, address := range path {
		parent := &views[len(views)-1]
		s.refreshAgentNavigationRowsLocked(parent)
		found := false
		for _, r := range parent.Rows {
			if r.Address == address && r.CanOpen() {
				found = true
				break
			}
		}
		if !found || seen[address.ChildSessionID] || address.ParentSessionID != parent.SessionID {
			return nil, false
		}
		seen[address.ChildSessionID] = true
		if i < len(path)-1 {
			branch := n.branches[agentCatalogKey(n.view.Address, path[:i+1])]
			if branch == nil || !branch.expanded || branch.busy || !branch.view.catalog.ready {
				return nil, false
			}
			views = append(views, branch.view)
		}
	}
	return views, true
}
func (s *Store) AgentCatalogTree(token uint64) []AgentCatalogTreeRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || token != n.view.Token {
		return nil
	}
	var out []AgentCatalogTreeRow
	var walk func(AgentNavigationView, []AgentSessionAddress)
	walk = func(v AgentNavigationView, prefix []AgentSessionAddress) {
		if len(prefix) >= 64 {
			return
		}
		s.refreshAgentNavigationRowsLocked(&v)
		for _, row := range v.Rows {
			path := append(append([]AgentSessionAddress(nil), prefix...), row.Address)
			entry := AgentCatalogTreeRow{Row: row, Path: path, Depth: len(prefix)}
			b := n.branches[agentCatalogKey(n.view.Address, path)]
			_, valid := s.agentCatalogPathLocked(path)
			if b != nil && valid {
				entry.Expanded = b.expanded
				entry.Busy = b.busy
				entry.Error = b.error
			}
			out = append(out, entry)
			if entry.Expanded && !entry.Busy && b.view.catalog.ready {
				walk(b.view, path)
			}
		}
	}
	walk(n.view, nil)
	return out
}
func (s *Store) ToggleAgentCatalogBranch(token uint64, path []AgentSessionAddress) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token {
		s.mu.Unlock()
		return
	}
	views, ok := s.agentCatalogPathLocked(path)
	if !ok {
		s.mu.Unlock()
		return
	}
	key := agentCatalogKey(n.view.Address, path)
	if n.branches == nil {
		n.branches = map[string]*agentCatalogBranch{}
	}
	b := n.branches[key]
	if b != nil && b.expanded {
		b.expanded = false
		b.request++
		if b.cancel != nil {
			b.cancel()
			b.cancel = nil
		}
		b.busy = false
		s.mu.Unlock()
		s.fire()
		return
	}
	if b == nil {
		b = &agentCatalogBranch{}
		n.branches[key] = b
	}
	address := path[len(path)-1]
	b.expanded = true
	b.busy = true
	b.error = ""
	b.request++
	b.view.SessionID = address.ChildSessionID
	b.view.Title = address.ChildSessionID
	for _, r := range views[len(views)-1].Rows {
		if r.Address == address {
			b.view.Title = r.Label
			break
		}
	}
	b.view.Address = address
	request := b.request
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	b.cancel = cancel
	client, generation, base := n.client, n.generation, n.view.Address
	captured := append([]AgentSessionAddress(nil), path...)
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		var projection agentNavigationProjection
		err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": address.ChildSessionID}}, &projection)
		s.mu.Lock()
		n := &s.agentNavigation
		if !s.agentNavigationOwnedLocked() || n.client != client || n.generation != generation || n.view.Address != base || n.branches[key] != b || b.request != request || !b.expanded {
			s.mu.Unlock()
			return
		}
		if _, ok := s.agentCatalogPathLocked(captured); !ok {
			b.expanded = false
			b.busy = false
			s.mu.Unlock()
			s.fire()
			return
		}
		if err == nil {
			err = seedAgentNavigationCatalog(&b.view, projection.Values, projection.AsOfSeq, s.items)
		}
		b.busy = false
		b.cancel = nil
		if err != nil {
			b.error = err.Error()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Entering a visible nested row constructs only its already-known parent path.
// The existing addressed history reader owns all subsequent live capabilities.
func (s *Store) OpenAgentCatalogPath(token uint64, path []AgentSessionAddress) {
	s.mu.Lock()
	n := &s.agentNavigation
	if !s.agentNavigationOwnedLocked() || n.view.Token != token || n.view.Busy || len(n.parents)+len(path) > 64 {
		s.mu.Unlock()
		return
	}
	views, ok := s.agentCatalogPathLocked(path)
	if !ok {
		s.mu.Unlock()
		return
	}
	address := path[len(path)-1]
	title := address.ChildSessionID
	for _, r := range views[len(views)-1].Rows {
		if r.Address == address {
			title = r.Label
			break
		}
	}
	for _, v := range views {
		v.Depth = len(n.parents)
		n.parents = append(n.parents, freezeAgentNavigationView(v))
	}
	n.view = AgentNavigationView{Token: token, SessionID: address.ChildSessionID, Title: title, Address: address, Depth: len(n.parents)}
	s.startAgentNavigationReadLocked(false)
	s.mu.Unlock()
	s.fire()
}

func (s *Store) cancelAgentCatalogReadsLocked() {
	for _, b := range s.agentNavigation.branches {
		if b.busy {
			if b.cancel != nil {
				b.cancel()
				b.cancel = nil
			}
			b.request++
			b.busy = false
			b.expanded = false
		}
	}
}

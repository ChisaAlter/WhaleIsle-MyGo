package session

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"time"
	"whaleisle/internal/dsh/api"
)

type WorkspaceLayoutState struct {
	SessionOrders   map[string][]string
	HostGeneration  uint64
	PreferenceError string
	GroupBy         string
	OrderBy         string
	Collapsed       map[string]bool
	Reordering      bool
	Error           string
}
type workspaceLayoutState struct {
	view    WorkspaceLayoutState
	request uint64
}

func (s *Store) workspaceLayoutLocked() WorkspaceLayoutState {
	out := s.workspaceLayout.view
	out.SessionOrders = cloneSessionOrders(out.SessionOrders)
	out.HostGeneration = s.eventsGeneration
	if out.GroupBy == "" {
		out.GroupBy = "workspace"
	}
	if out.OrderBy == "" {
		out.OrderBy = "manual"
	}
	out.Collapsed = map[string]bool{}
	for id, v := range s.workspaceLayout.view.Collapsed {
		out.Collapsed[id] = v
	}
	return out
}
func (s *Store) SetWorkspaceGrouping(mode string) {
	if mode != "workspace" && mode != "workspace-tree" && mode != "flat" {
		return
	}
	s.mu.Lock()
	s.workspaceLayout.view.GroupBy = mode
	s.workspacePreferencesChangedLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetWorkspaceOrdering(mode string) {
	if mode != "manual" && mode != "updated" {
		return
	}
	s.mu.Lock()
	s.workspaceLayout.view.OrderBy = mode
	s.workspacePreferencesChangedLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ToggleWorkspaceGroup(id string) {
	s.mu.Lock()
	if s.workspaceLayout.view.Collapsed == nil {
		s.workspaceLayout.view.Collapsed = map[string]bool{}
	}
	s.workspaceLayout.view.Collapsed[id] = !s.workspaceLayout.view.Collapsed[id]
	s.workspacePreferencesChangedLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) resetWorkspaceLayoutHostLocked() {
	s.resetWorkspacePreferencesHostLocked()
	s.workspaceLayout.request++
	s.workspaceLayout.view.Collapsed = nil
	s.workspaceLayout.view.Reordering = false
	s.workspaceLayout.view.Error = ""
}
func (s *Store) reorderWorkspace(ctx context.Context, id, beforeID string) {
	s.reorderWorkspaceForHost(ctx, 0, id, beforeID)
}
func (s *Store) reorderWorkspaceForHost(ctx context.Context, expectedGeneration uint64, id, beforeID string) {
	id = strings.TrimSpace(id)
	beforeID = strings.TrimSpace(beforeID)
	s.mu.Lock()
	client, generation := s.client, s.eventsGeneration
	if (expectedGeneration != 0 && expectedGeneration != generation) || client == nil || !s.workspaceReady || s.workspaceLayout.view.Reordering || id == "" || id == beforeID || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if _, ok := s.workspaceMemberships[id]; !ok {
		s.mu.Unlock()
		return
	}
	if beforeID != "" {
		if _, ok := s.workspaceMemberships[beforeID]; !ok {
			s.mu.Unlock()
			return
		}
	}
	s.workspaceLayout.request++
	token := s.workspaceLayout.request
	s.workspaceLayout.view.Reordering = true
	s.workspaceLayout.view.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		req := api.WorkspaceInsertBeforeRequest{WorkspaceId: id}
		if beforeID != "" {
			req.BeforeWorkspaceId = &beforeID
		}
		var out api.WorkspaceOrderValue
		err := client.Call(callCtx, "workspace/insertBefore", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || s.workspaceLayout.request != token {
			s.mu.Unlock()
			return
		}
		s.workspaceLayout.view.Reordering = false
		if err == nil && out.WorkspaceIds == nil {
			err = errors.New("Host returned no workspace order")
		}
		if err != nil {
			s.workspaceLayout.view.Error = err.Error()
		} else {
			s.workspaceOrder = append([]string(nil), out.WorkspaceIds...)
			s.signalWorkspaceLocked()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// WorkspaceGroup always uses Host membership, never guesses membership from cwd.
// ParentID is a registered ancestor, derived lexically from canonical Host paths.
type WorkspaceGroup struct {
	Key       string
	Kind      string
	Workspace WorkspaceRow
	ParentID  string
	Depth     int
	Items     []Item
}
type WorkspaceSessionRow struct {
	Item  Item
	Depth int
}

func hostPathKey(p string) string {
	p = strings.TrimSpace(p)
	windows := strings.Contains(p, `\`) || (len(p) >= 2 && p[1] == ':')
	if windows {
		p = strings.ToLower(strings.ReplaceAll(p, `\`, "/"))
	}
	if p == "" {
		return ""
	}
	return path.Clean(p)
}
func registeredWorkspaceParents(rows []WorkspaceRow) map[string]string {
	parents := map[string]string{}
	for _, row := range rows {
		child := hostPathKey(row.Path)
		longest := 0
		for _, candidate := range rows {
			if candidate.WorkspaceID == row.WorkspaceID {
				continue
			}
			parent := hostPathKey(candidate.Path)
			prefix := strings.TrimRight(parent, "/") + "/"
			if parent != "" && child != parent && strings.HasPrefix(child, prefix) && len(parent) > longest {
				parents[row.WorkspaceID] = candidate.WorkspaceID
				longest = len(parent)
			}
		}
	}
	return parents
}
func WorkspaceSessionOwners(workspaces []WorkspaceRow, items []Item) map[string]string {
	owners := map[string]string{}
	byID := map[string]Item{}
	for _, ws := range workspaces {
		for _, id := range ws.SessionIDs {
			if owners[id] == "" {
				owners[id] = ws.WorkspaceID
			}
		}
	}
	for _, item := range items {
		byID[item.SessionID] = item
	}
	for _, item := range items {
		if owners[item.SessionID] != "" {
			continue
		}
		seen := map[string]bool{}
		for id := item.ParentID; id != "" && !seen[id]; id = byID[id].ParentID {
			seen[id] = true
			if owners[id] != "" {
				owners[item.SessionID] = owners[id]
				break
			}
		}
	}
	return owners
}
func BuildWorkspaceGroups(snap ViewState, items []Item) []WorkspaceGroup {
	if snap.Items == nil {
		snap.Items = items
	}
	if snap.WorkspaceLayout.GroupBy == "flat" {
		order := localSessionOrder(snap, "::flat")
		return []WorkspaceGroup{{Key: "::flat", Kind: "flat", Workspace: WorkspaceRow{SessionIDs: order}, Items: append([]Item(nil), items...)}}
	}
	owners := WorkspaceSessionOwners(snap.Workspaces, items)
	knownWorkspace := map[string]bool{}
	for _, ws := range snap.Workspaces {
		knownWorkspace[ws.WorkspaceID] = true
	}
	byGroup := map[string][]Item{}
	var scratch []Item
	for _, item := range items {
		if id := owners[item.SessionID]; id != "" && knownWorkspace[id] {
			byGroup[id] = append(byGroup[id], item)
		} else {
			// The Original has one unfiled bucket ("无工作目录"); a session
			// whose folder is not a registered workspace belongs there too.
			scratch = append(scratch, item)
		}
	}
	parents := map[string]string{}
	if snap.WorkspaceLayout.GroupBy == "workspace-tree" {
		parents = registeredWorkspaceParents(snap.Workspaces)
	}
	var out []WorkspaceGroup
	visited := map[string]bool{}
	var emit func(string, int)
	emit = func(parent string, depth int) {
		for _, ws := range snap.Workspaces {
			if visited[ws.WorkspaceID] || parents[ws.WorkspaceID] != parent {
				continue
			}
			visited[ws.WorkspaceID] = true
			out = append(out, WorkspaceGroup{Key: ws.WorkspaceID, Kind: "workspace", Workspace: ws, ParentID: parent, Depth: depth, Items: byGroup[ws.WorkspaceID]})
			emit(ws.WorkspaceID, depth+1)
		}
	}
	emit("", 0)
	if len(scratch) > 0 {
		out = append(out, WorkspaceGroup{Key: "::scratch", Kind: "scratch", Workspace: WorkspaceRow{SessionIDs: localSessionOrder(snap, "::scratch")}, Items: scratch})
	}
	return out
}
func OrderedWorkspaceSessionRows(items []Item, manual, pinned []string, order string, flat bool) []WorkspaceSessionRow {
	items = append([]Item(nil), items...)
	ranks := map[string]int{}
	for i, id := range manual {
		ranks[id] = i
	}
	less := func(a, b Item) bool {
		pa, pb := PinRank(a.SessionID, pinned), PinRank(b.SessionID, pinned)
		if pa >= 0 || pb >= 0 {
			if pa < 0 {
				return false
			}
			if pb < 0 {
				return true
			}
			if pa != pb {
				return pa < pb
			}
		}
		if order != "updated" {
			ra, oka := ranks[a.SessionID]
			rb, okb := ranks[b.SessionID]
			if oka != okb {
				return oka
			}
			if oka && ra != rb {
				return ra < rb
			}
		}
		if a.UpdatedAt != b.UpdatedAt {
			return a.UpdatedAt > b.UpdatedAt
		}
		return a.SessionID < b.SessionID
	}
	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })
	if flat {
		out := make([]WorkspaceSessionRow, 0, len(items))
		for _, it := range items {
			out = append(out, WorkspaceSessionRow{Item: it})
		}
		return out
	}
	byID := map[string]Item{}
	children := map[string][]Item{}
	for _, it := range items {
		byID[it.SessionID] = it
	}
	var roots []Item
	for _, it := range items {
		if _, ok := byID[it.ParentID]; it.ParentID == "" || !ok {
			roots = append(roots, it)
		} else {
			children[it.ParentID] = append(children[it.ParentID], it)
		}
	}
	seen := map[string]bool{}
	var out []WorkspaceSessionRow
	var walk func(Item, int)
	walk = func(it Item, depth int) {
		if seen[it.SessionID] {
			return
		}
		seen[it.SessionID] = true
		out = append(out, WorkspaceSessionRow{Item: it, Depth: depth})
		for _, child := range children[it.SessionID] {
			walk(child, depth+1)
		}
	}
	for _, it := range roots {
		walk(it, 0)
	}
	for _, it := range items {
		walk(it, 0)
	} // corrupt parent cycles remain visible once
	return out
}

// WorkspaceMoveAnchor is the same insert-before operation for arrow and drag
// affordances. Tree mode limits movement to the target's registered siblings.
func WorkspaceMoveAnchor(groups []WorkspaceGroup, id string, delta int) (string, bool) {
	parent := ""
	found := false
	for _, g := range groups {
		if g.Key == id && g.Kind == "workspace" {
			parent = g.ParentID
			found = true
			break
		}
	}
	if !found {
		return "", false
	}
	var siblings []string
	index := -1
	for _, g := range groups {
		if g.Kind == "workspace" && g.ParentID == parent {
			if g.Key == id {
				index = len(siblings)
			}
			siblings = append(siblings, g.Key)
		}
	}
	if delta < 0 && index > 0 {
		return siblings[index-1], true
	}
	if delta > 0 && index >= 0 && index+1 < len(siblings) {
		if index+2 < len(siblings) {
			return siblings[index+2], true
		}
		return "", true
	}
	return "", false
}

// BeginWorkspaceManagement addresses the clicked registry row, independently
// of the currently selected workspace or session.
func (s *Store) BeginWorkspaceManagement(id string) { s.beginWorkspaceManagement("manage", id) }

func (s *Store) reorderWorkspaceSession(ctx context.Context, workspaceID, sessionID, beforeID string) {
	workspaceID = strings.TrimSpace(workspaceID)
	sessionID = strings.TrimSpace(sessionID)
	beforeID = strings.TrimSpace(beforeID)
	s.mu.Lock()
	client, generation := s.client, s.eventsGeneration
	ws, exists := s.workspaceMemberships[workspaceID]
	member, anchor := false, beforeID == ""
	for _, id := range ws.SessionIDs {
		if id == sessionID {
			member = true
		}
		if id == beforeID {
			anchor = true
		}
	}
	if client == nil || !exists || !member || !anchor || sessionID == beforeID || s.workspaceLayout.view.Reordering || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.workspaceLayout.request++
	token := s.workspaceLayout.request
	s.workspaceLayout.view.Reordering = true
	s.workspaceLayout.view.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		req := api.WorkspaceInsertSessionBeforeRequest{WorkspaceId: workspaceID, SessionId: sessionID}
		if beforeID != "" {
			req.BeforeSessionId = &beforeID
		}
		var out struct {
			Workspace api.WorkspaceCreateValueWorkspace `json:"workspace"`
		}
		err := client.Call(callCtx, "workspace/insertSessionBefore", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || s.workspaceLayout.request != token {
			s.mu.Unlock()
			return
		}
		s.workspaceLayout.view.Reordering = false
		if err == nil && (out.Workspace.WorkspaceId != workspaceID || out.Workspace.Path == "") {
			err = errors.New("Host returned an incomplete workspace session order")
		}
		if err != nil {
			s.workspaceLayout.view.Error = err.Error()
		} else {
			s.putWorkspaceRowLocked(workspaceRowFromCreate(out.Workspace))
			s.lastAction = "session reordered"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

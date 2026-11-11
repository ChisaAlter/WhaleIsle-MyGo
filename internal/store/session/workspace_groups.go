package session

import (
	"context"
	"errors"
	"path"
	"slices"
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
	ArchivedFilter  string
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
		out.OrderBy = "updated"
	}
	if out.ArchivedFilter != "show" && out.ArchivedFilter != "only" {
		out.ArchivedFilter = "default"
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
	s.workspacePreferencesChangedLocked("group", "")
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetWorkspaceOrdering(mode string) {
	if mode != "manual" && mode != "updated" {
		return
	}
	s.mu.Lock()
	changed := s.setWorkspaceOrderingLocked(mode)
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) ToggleWorkspaceGroup(id string) {
	s.mu.Lock()
	if s.workspaceLayout.view.Collapsed == nil {
		s.workspaceLayout.view.Collapsed = map[string]bool{}
	}
	snap := ViewState{Items: s.items, SelectedID: s.selectedID, ScratchCwd: s.scratchCwd,
		Workspaces: s.workspaceRowsLocked(), WorkspaceMembers: s.workspaceMembersLocked(), WorkspaceLayout: s.workspaceLayoutLocked()}
	s.workspaceLayout.view.Collapsed[id] = !WorkspaceGroupCollapsed(snap, id)
	s.workspacePreferencesChangedLocked("collapsed", id)
	s.mu.Unlock()
	s.fire()
}
func (s *Store) resetWorkspaceLayoutHostLocked() {
	s.invalidateViewLocked()
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

const PinnedWorkspaceGroup = "::pinned"

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
func buildWorkspaceGroups(snap ViewState, items []Item) []WorkspaceGroup {
	return buildWorkspaceGroupsForSidebar(snap, items, false)
}

func buildWorkspaceGroupsForSidebar(snap ViewState, items []Item, hoistPinned bool) []WorkspaceGroup {
	if snap.Items == nil {
		snap.Items = items
	}
	// Accounts come from complete Host membership, never parent lineage or cwd.
	// Hidden and missing rows retain their browser-local manual-order slots.
	owners := map[string]string{}
	for _, ws := range snap.Workspaces {
		for _, id := range ws.SessionIDs {
			if _, accounted := owners[id]; !accounted {
				owners[id] = ws.WorkspaceID
			}
		}
	}
	listed := func(item Item) bool {
		if item.SessionID == "" || item.Origin == "subagent" || item.Blank && item.SessionID != snap.SelectedID {
			return false
		}
		if item.Presentation != nil && item.Presentation.Owner != "" && item.Presentation.Owner != "whaleisle" {
			return false
		}
		_, member := owners[item.SessionID]
		return member || snap.IsNoDirectory(item)
	}
	visible := make([]Item, 0, len(items))
	for _, item := range items {
		if listed(item) {
			visible = append(visible, item)
		}
	}
	orderIndex := newWorkspaceOrderIndex(snap)
	accountOrder := func(memberIDs []string, key string) []string {
		return orderIndex.order(snap, memberIDs, key)
	}
	var flatMembers, scratchMembers []string
	for _, item := range snap.Items {
		if listed(item) {
			flatMembers = append(flatMembers, item.SessionID)
		}
		if _, member := owners[item.SessionID]; !member {
			scratchMembers = append(scratchMembers, item.SessionID)
		}
	}
	if snap.WorkspaceLayout.GroupBy == "flat" {
		return []WorkspaceGroup{{Key: "::flat", Kind: "flat", Workspace: WorkspaceRow{SessionIDs: accountOrder(flatMembers, "::flat")}, Items: visible}}
	}
	byGroup := map[string][]Item{}
	var scratch []Item
	pinned := map[string]Item{}
	for _, item := range visible {
		if hoistPinned && snap.WorkspaceLayout.ArchivedFilter != "only" && !item.Blank &&
			slices.Contains(snap.PinnedSessionIDs, item.SessionID) && !slices.Contains(snap.ArchivedSessionIDs, item.SessionID) {
			pinned[item.SessionID] = item
			continue
		}
		if id, member := owners[item.SessionID]; member {
			byGroup[id] = append(byGroup[id], item)
		} else {
			scratch = append(scratch, item)
		}
	}
	parents := map[string]string{}
	if snap.WorkspaceLayout.GroupBy == "workspace-tree" {
		parents = registeredWorkspaceParents(snap.Workspaces)
	}
	var out []WorkspaceGroup
	if len(pinned) > 0 {
		group := WorkspaceGroup{Key: PinnedWorkspaceGroup, Kind: "pinned"}
		for _, id := range snap.PinnedSessionIDs {
			if item, ok := pinned[id]; ok {
				group.Items = append(group.Items, item)
				group.Workspace.SessionIDs = append(group.Workspace.SessionIDs, id)
				delete(pinned, id)
			}
		}
		out = append(out, group)
	}
	visited := map[string]bool{}
	var emit func(string, int)
	emit = func(parent string, depth int) {
		for _, ws := range snap.Workspaces {
			if visited[ws.WorkspaceID] || parents[ws.WorkspaceID] != parent {
				continue
			}
			visited[ws.WorkspaceID] = true
			ws.SessionIDs = accountOrder(ws.SessionIDs, ws.WorkspaceID)
			out = append(out, WorkspaceGroup{Key: ws.WorkspaceID, Kind: "workspace", Workspace: ws, ParentID: parent, Depth: depth, Items: byGroup[ws.WorkspaceID]})
			emit(ws.WorkspaceID, depth+1)
		}
	}
	emit("", 0)
	if len(scratch) > 0 {
		out = append(out, WorkspaceGroup{Key: "::scratch", Kind: "scratch", Workspace: WorkspaceRow{SessionIDs: accountOrder(scratchMembers, "::scratch")}, Items: scratch})
	}
	return out
}
func OrderedWorkspaceSessionRows(items []Item, manual, pinned []string, order string, flat bool) []WorkspaceSessionRow {
	items = append([]Item(nil), items...)
	ranks := map[string]int{}
	for i, id := range manual {
		if _, present := ranks[id]; !present {
			ranks[id] = i
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if order == "manual" {
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
	})
	// sectionMembers keeps each partition's account order. ParentID is fork
	// metadata, not indentation or a reason to move saved rows.
	out := make([]WorkspaceSessionRow, 0, len(items))
	for _, partition := range []int{0, 1, 2} {
		for _, item := range items {
			part := 2
			if item.Blank {
				part = 0
			} else if PinRank(item.SessionID, pinned) >= 0 {
				part = 1
			}
			if part == partition {
				out = append(out, WorkspaceSessionRow{Item: item})
			}
		}
	}
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
	s.mu.Lock()
	changed := false
	if ctx.Err() == nil {
		changed = s.insertWorkspaceSessionLocalLocked(s.eventsGeneration, workspaceID, sessionID, beforeID)
	}
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}

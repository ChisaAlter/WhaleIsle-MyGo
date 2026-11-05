package session

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"whaleisle/internal/dsh/api"
)

// WorkspaceRow is the Host registry projection. Paths belong to that Host and
// must never be validated with the desktop's local filesystem.
type WorkspaceRow struct {
	WorkspaceID string   `json:"workspaceId"`
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	SessionIDs  []string `json:"sessionIds"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// WorkspaceNavigationState is one token-bound picker or management dialog.
type WorkspaceNavigationState struct {
	Token       uint64
	Mode        string // pick, rename, remove
	TargetID    string
	TargetTitle string
	Draft       string
	Busy        bool
	Error       string
	Listing     *api.DirectoryListing
	ListBusy    bool
}
type workspaceNavigationOperation struct {
	view            WorkspaceNavigationState
	generation      uint64
	cwd, selectedID string
	cancel          context.CancelFunc
	listToken       uint64
}

func (s *Store) invalidateWorkspaceNavigationLocked() {
	s.invalidateViewLocked()
	if s.workspaceNavigation.cancel != nil {
		s.workspaceNavigation.cancel()
	}
	s.workspaceNavigationToken++
	s.workspaceNavigation = workspaceNavigationOperation{}
}
func (s *Store) workspaceNavigationCurrentLocked(token uint64) bool {
	op := &s.workspaceNavigation
	return op.view.Mode != "" && op.view.Token == token && s.client != nil && op.generation == s.eventsGeneration && op.cwd == s.cwd && op.selectedID == s.selectedID
}
func (s *Store) workspaceNavigationSnapshotLocked() *WorkspaceNavigationState {
	if s.workspaceNavigation.view.Mode == "" {
		return nil
	}
	if !s.workspaceNavigationCurrentLocked(s.workspaceNavigation.view.Token) {
		s.invalidateWorkspaceNavigationLocked()
		return nil
	}
	out := s.workspaceNavigation.view
	if out.Listing != nil {
		copy := *out.Listing
		copy.Crumbs = append([]api.DirectoryListingCrumbsItem(nil), copy.Crumbs...)
		copy.Entries = append([]api.DirectoryListingCrumbsItem(nil), copy.Entries...)
		out.Listing = &copy
	}
	return &out
}
func (s *Store) workspaceRowsLocked() []WorkspaceRow {
	rows := make([]WorkspaceRow, 0, len(s.workspaceMemberships))
	seen := map[string]bool{}
	add := func(id string) {
		if ws, ok := s.workspaceMemberships[id]; ok && !seen[id] {
			row := WorkspaceRow(ws)
			row.SessionIDs = append([]string(nil), row.SessionIDs...)
			rows = append(rows, row)
			seen[id] = true
		}
	}
	for _, id := range s.workspaceOrder {
		add(id)
	}
	// Deterministic fallback for legacy baselines and partial follow increments.
	var remaining []string
	for id := range s.workspaceMemberships {
		if !seen[id] {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	for _, id := range remaining {
		add(id)
	}
	return rows
}
func (s *Store) openWorkspaceNavigationLocked(mode, id string) {
	s.invalidateViewLocked()
	s.invalidateWorkspaceNavigationLocked()
	op := workspaceNavigationOperation{view: WorkspaceNavigationState{Token: s.workspaceNavigationToken, Mode: mode, TargetID: id}, generation: s.eventsGeneration, cwd: s.cwd, selectedID: s.selectedID}
	if ws, ok := s.workspaceMemberships[id]; ok {
		op.view.TargetTitle = ws.Title
		if op.view.TargetTitle == "" {
			op.view.TargetTitle = ws.Path
		}
		if mode == "rename" {
			op.view.Draft = op.view.TargetTitle
		}
	}
	s.workspaceNavigation = op
}
func (s *Store) OpenWorkspacePicker() {
	s.mu.Lock()
	if s.client == nil {
		if s.lastAction != "Connect a Host to choose a registered workspace" {
			s.invalidateViewLocked()
		}
		s.lastAction = "Connect a Host to choose a registered workspace"
		s.mu.Unlock()
		s.fire()
		return
	}
	if s.workspaceNavigation.view.Busy {
		s.mu.Unlock()
		return
	}
	s.openWorkspaceNavigationLocked("pick", "")
	s.mu.Unlock()
	s.fire()
}
func (s *Store) BeginWorkspaceRename(id string)  { s.beginWorkspaceManagement("rename", id) }
func (s *Store) BeginWorkspaceRemoval(id string) { s.beginWorkspaceManagement("remove", id) }
func (s *Store) beginWorkspaceManagement(mode, id string) {
	s.mu.Lock()
	if s.client == nil || !s.workspaceReady {
		s.mu.Unlock()
		return
	}
	if _, ok := s.workspaceMemberships[id]; !ok {
		s.mu.Unlock()
		return
	}
	if s.workspaceNavigation.view.Busy {
		s.mu.Unlock()
		return
	}
	s.openWorkspaceNavigationLocked(mode, id)
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelWorkspaceNavigation(token uint64) {
	s.mu.Lock()
	if s.workspaceNavigation.view.Token != token {
		s.mu.Unlock()
		return
	}
	s.invalidateWorkspaceNavigationLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetWorkspaceNavigationDraft(token uint64, draft string) {
	s.mu.Lock()
	if s.workspaceNavigationCurrentLocked(token) && !s.workspaceNavigation.view.Busy {
		if s.workspaceNavigation.view.Draft != draft || s.workspaceNavigation.view.Error != "" {
			s.invalidateViewLocked()
		}
		s.workspaceNavigation.view.Draft = draft
		s.workspaceNavigation.view.Error = ""
	}
	s.mu.Unlock()
}
func (s *Store) SetWorkspaceNavigationError(token uint64, err error) {
	s.mu.Lock()
	if s.workspaceNavigationCurrentLocked(token) && err != nil {
		if s.workspaceNavigation.view.Error != err.Error() {
			s.invalidateViewLocked()
		}
		s.workspaceNavigation.view.Error = err.Error()
	}
	s.mu.Unlock()
	s.fire()
}

// persist is called under the Store lock immediately before selection commits.
// It must only save config, must not call Store methods, and may reject the commit.
func (s *Store) commitWorkspaceLocked(row WorkspaceRow, persist func(string) error) error {
	if strings.TrimSpace(row.WorkspaceID) == "" || strings.TrimSpace(row.Path) == "" {
		return errors.New("Host returned an incomplete workspace")
	}
	if persist != nil {
		if err := persist(row.Path); err != nil {
			return err
		}
	}
	s.invalidateWorkspaceNavigationLocked()
	s.cwd, s.workspaceID = row.Path, row.WorkspaceID
	// The previously selected session owns its old cwd; leave it before rendering
	// Files/composer for the new workspace. Drafts remain owned by their sessions.
	s.selectSessionLocked("")
	s.lastAction = "Workspace selected: " + row.Title
	return nil
}
func (s *Store) ChooseRegisteredWorkspace(token uint64, id string, persist func(string) error) bool {
	s.mu.Lock()
	if !s.workspaceNavigationCurrentLocked(token) || s.workspaceNavigation.view.Mode != "pick" || s.workspaceNavigation.view.Busy || !s.workspaceReady {
		s.mu.Unlock()
		return false
	}
	ws, ok := s.workspaceMemberships[id]
	if !ok {
		if s.workspaceNavigation.view.Error != "Workspace is no longer registered" {
			s.invalidateViewLocked()
		}
		s.workspaceNavigation.view.Error = "Workspace is no longer registered"
		s.mu.Unlock()
		s.fire()
		return false
	}
	err := s.commitWorkspaceLocked(WorkspaceRow(ws), persist)
	if err != nil {
		if s.workspaceNavigation.view.Error != err.Error() {
			s.invalidateViewLocked()
		}
		s.workspaceNavigation.view.Error = err.Error()
	}
	s.mu.Unlock()
	s.fire()
	return err == nil
}
func workspaceRowFromCreate(ws api.WorkspaceCreateValueWorkspace) WorkspaceRow {
	return WorkspaceRow{WorkspaceID: ws.WorkspaceId, Path: ws.Path, Title: ws.Title, SessionIDs: append([]string(nil), ws.SessionIds...), CreatedAt: ws.CreatedAt, UpdatedAt: ws.UpdatedAt}
}
func (s *Store) putWorkspaceRowLocked(row WorkspaceRow) {
	s.invalidateViewLocked()
	if s.workspaceMemberships == nil {
		s.workspaceMemberships = map[string]workspaceMembership{}
	}
	if _, ok := s.workspaceMemberships[row.WorkspaceID]; !ok {
		s.workspaceOrder = append(s.workspaceOrder, row.WorkspaceID)
	}
	s.workspaceMemberships[row.WorkspaceID] = workspaceMembership(row)
	s.signalWorkspaceLocked()
}

// RegisterWorkspace keeps the pick and draft open on any Host/config error.
// The path is opaque until workspace/create returns its authoritative row.
func (s *Store) RegisterWorkspace(ctx context.Context, token uint64, path string, persist func(string) error, committed func(WorkspaceRow)) {
	path = strings.TrimSpace(path)
	s.mu.Lock()
	if !s.workspaceNavigationCurrentLocked(token) || s.workspaceNavigation.view.Mode != "pick" || s.workspaceNavigation.view.Busy || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if s.workspaceNavigation.view.Draft != path {
		s.invalidateViewLocked()
	}
	s.workspaceNavigation.view.Draft = path
	if path == "" || !s.workspaceReady {
		if s.workspaceNavigation.view.Error != "Choose a directory after the Host workspace list is ready" {
			s.invalidateViewLocked()
		}
		s.workspaceNavigation.view.Error = "Choose a directory after the Host workspace list is ready"
		s.mu.Unlock()
		s.fire()
		return
	}
	client := s.client
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	s.workspaceNavigation.cancel = cancel
	s.workspaceNavigation.view.Busy = true
	s.workspaceNavigation.view.Error = ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		var out api.WorkspaceCreateValue
		err := client.Call(callCtx, "workspace/create", map[string]any{"request": map[string]any{"path": path}}, &out)
		row := workspaceRowFromCreate(out.Workspace)
		s.mu.Lock()
		if !s.workspaceNavigationCurrentLocked(token) || s.client != client {
			s.mu.Unlock()
			return
		}
		if err == nil && (row.WorkspaceID == "" || row.Path == "") {
			err = errors.New("Host returned an incomplete workspace")
		}
		if err == nil {
			s.putWorkspaceRowLocked(row)
			err = s.commitWorkspaceLocked(row, persist)
		}
		if err != nil {
			s.workspaceNavigation.view.Busy = false
			s.workspaceNavigation.view.Error = err.Error()
			s.workspaceNavigation.cancel = nil
			s.invalidateViewLocked()
		}
		s.mu.Unlock()
		s.fire()
		if err == nil && committed != nil {
			committed(row)
		}
	}()
}

func (s *Store) ConfirmWorkspaceRename(ctx context.Context, token uint64, title string) {
	title = strings.TrimSpace(title)
	s.mu.Lock()
	if !s.workspaceNavigationCurrentLocked(token) || s.workspaceNavigation.view.Mode != "rename" || s.workspaceNavigation.view.Busy || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	op := &s.workspaceNavigation
	if op.view.Draft != title {
		s.invalidateViewLocked()
	}
	op.view.Draft = title
	ws, exists := s.workspaceMemberships[op.view.TargetID]
	problem := ""
	if !exists {
		problem = "Workspace is no longer registered"
	} else if title == "" {
		problem = "Enter a workspace name"
	} else if title == ws.Title {
		problem = "Enter a different workspace name"
	}
	for id, other := range s.workspaceMemberships {
		if id != op.view.TargetID && other.Title == title {
			problem = "Another workspace already uses that name"
		}
	}
	if problem != "" {
		if op.view.Error != problem {
			s.invalidateViewLocked()
		}
		op.view.Error = problem
		s.mu.Unlock()
		s.fire()
		return
	}
	client, id := s.client, op.view.TargetID
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	op.cancel = cancel
	op.view.Busy = true
	op.view.Error = ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		var out struct {
			Workspace api.WorkspaceCreateValueWorkspace `json:"workspace"`
		}
		err := client.Call(callCtx, "workspace/rename", map[string]any{"request": api.WorkspaceRenameRequest{WorkspaceId: id, Title: title}}, &out)
		row := workspaceRowFromCreate(out.Workspace)
		s.mu.Lock()
		if !s.workspaceNavigationCurrentLocked(token) || s.client != client {
			s.mu.Unlock()
			return
		}
		if err == nil && (row.WorkspaceID != id || row.Path == "") {
			err = errors.New("Host returned an incomplete renamed workspace")
		}
		if err != nil {
			s.workspaceNavigation.view.Busy = false
			s.workspaceNavigation.view.Error = err.Error()
			s.workspaceNavigation.cancel = nil
			s.invalidateViewLocked()
		} else {
			s.putWorkspaceRowLocked(row)
			s.invalidateWorkspaceNavigationLocked()
			s.lastAction = "Workspace renamed: " + row.Title
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) ConfirmWorkspaceRemoval(ctx context.Context, token uint64) {
	s.mu.Lock()
	if !s.workspaceNavigationCurrentLocked(token) || s.workspaceNavigation.view.Mode != "remove" || s.workspaceNavigation.view.Busy || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	op := &s.workspaceNavigation
	client, id := s.client, op.view.TargetID
	workspace, registered := s.workspaceMemberships[id]
	if !registered {
		s.invalidateWorkspaceNavigationLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	// Match ui-workspace.deleteWorkspace: removing this registration closes
	// its current conversation only after both the receipt and list echo land.
	// Capture membership before the authoritative remove frame discards it.
	selected := s.selectedID
	closeSelected := false
	for _, member := range workspace.SessionIDs {
		if member == selected && selected != "" {
			closeSelected = true
			break
		}
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	op.cancel = cancel
	op.view.Busy = true
	op.view.Error = ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		var out api.WorkspaceDeleteValue
		err := client.Call(callCtx, "workspace/delete", map[string]any{"request": api.WorkspaceDeleteRequest{WorkspaceId: id}}, &out)
		if err == nil && !out.Deleted {
			err = errors.New("Host did not remove the workspace registration")
		}
		// Receipt and follow projection can arrive in either order. Keep confirmation
		// pending until both have landed, avoiding one stale registry frame.
		for {
			s.mu.Lock()
			if !s.workspaceNavigationCurrentLocked(token) || s.client != client {
				s.mu.Unlock()
				return
			}
			if err != nil {
				s.workspaceNavigation.view.Busy = false
				s.workspaceNavigation.view.Error = err.Error()
				s.workspaceNavigation.cancel = nil
				s.invalidateViewLocked()
				s.mu.Unlock()
				s.fire()
				return
			}
			if _, exists := s.workspaceMemberships[id]; !exists {
				if s.workspaceID == id {
					s.workspaceID = ""
				}
				if closeSelected && s.selectedID == selected {
					s.selectSessionLocked("")
				}
				s.invalidateWorkspaceNavigationLocked()
				s.lastAction = "Workspace registration removed; files are unchanged"
				s.mu.Unlock()
				s.fire()
				return
			}
			if s.workspaceErr != nil {
				err = errors.New("Registration removed, but the Host list could not refresh; reconnect to confirm")
				s.mu.Unlock()
				continue
			}
			changed := s.workspaceChanged
			s.mu.Unlock()
			select {
			case <-changed:
			case <-callCtx.Done():
				err = errors.New("Registration removed, but the updated Host list has not arrived; reconnect to confirm")
			}
		}
	}()
}
func (s *Store) removeWorkspaceOrderLocked(id string) {
	s.invalidateViewLocked()
	out := s.workspaceOrder[:0]
	for _, candidate := range s.workspaceOrder {
		if candidate != id {
			out = append(out, candidate)
		}
	}
	s.workspaceOrder = out
}

// BrowseWorkspaceDirectories uses the same Host as the registered-workspace list.
// Listing replies are scoped to the dialog, Host, selection and newest navigation.
func (s *Store) BrowseWorkspaceDirectories(ctx context.Context, token uint64, path string) {
	s.mu.Lock()
	if !s.workspaceNavigationCurrentLocked(token) || s.workspaceNavigation.view.Mode != "pick" || s.workspaceNavigation.view.Busy {
		s.mu.Unlock()
		return
	}
	s.workspaceNavigation.listToken++
	request := s.workspaceNavigation.listToken
	client := s.client
	if !s.workspaceNavigation.view.ListBusy || s.workspaceNavigation.view.Error != "" {
		s.invalidateViewLocked()
	}
	s.workspaceNavigation.view.ListBusy = true
	s.workspaceNavigation.view.Error = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		var listing api.DirectoryListing
		args := map[string]any{}
		if strings.TrimSpace(path) != "" {
			args["path"] = path
		}
		err := client.Call(callCtx, "directoryPicker/list", args, &listing)
		s.mu.Lock()
		if !s.workspaceNavigationCurrentLocked(token) || s.client != client || s.workspaceNavigation.listToken != request {
			s.mu.Unlock()
			return
		}
		s.workspaceNavigation.view.ListBusy = false
		if err != nil {
			s.workspaceNavigation.view.Error = err.Error()
		} else {
			s.workspaceNavigation.view.Listing = &listing
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
	}()
}

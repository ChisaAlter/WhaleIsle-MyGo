package pluginui

import (
	"context"
	"errors"
	"strconv"
)

type WorkspaceDirectoryAction struct {
	Handle    string            `json:"handle"`
	ID        string            `json:"id"`
	Label     string            `json:"label"`
	Order     int               `json:"order"`
	Picker    bool              `json:"picker"`
	RowLabels map[string]string `json:"rowLabels"`
}
type WorkspaceNavigation struct {
	SessionID        string `json:"sessionId"`
	Owner            string `json:"owner"`
	WorkingDirectory string `json:"workingDirectory"`
	Title            string `json:"title"`
	Archived         bool   `json:"archived"`
	CanRename        bool   `json:"canRename"`
	CanArchive       bool   `json:"canArchive"`
	CanRestore       bool   `json:"canRestore"`
}

func validWorkspaceProjection(actions []WorkspaceDirectoryAction, rows []WorkspaceNavigation) bool {
	if len(actions) > 512 || len(rows) > 8192 {
		return false
	}
	for _, a := range actions {
		if a.Handle == "" || len(a.Handle) > 64 || a.ID == "" || len(a.ID) > 8192 || len(a.Label) > 8192 || len(a.RowLabels) > 8192 {
			return false
		}
		for p, l := range a.RowLabels {
			if len(p) > 32768 || len(l) > 8192 {
				return false
			}
		}
	}
	for _, r := range rows {
		if r.SessionID == "" || r.Owner == "" || len(r.SessionID) > 8192 || len(r.Owner) > 8192 || len(r.WorkingDirectory) > 32768 || len(r.Title) > 8192 {
			return false
		}
	}
	return true
}

type workspaceOperation struct {
	page   *bridgePage
	result chan error
}
type workspaceOperationState struct {
	next    uint64
	pending map[string]*workspaceOperation
}

func (o *Owner) retireWorkspaceOperationsLocked() {
	for id, p := range o.workspaceOps.pending {
		p.result <- errors.New("plugin UI: workspace source changed")
		delete(o.workspaceOps.pending, id)
	}
}
func (o *Owner) finishWorkspaceOperation(page *bridgePage, id, message string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	p := o.workspaceOps.pending[id]
	if p == nil || p.page != page || o.page != page {
		return
	}
	delete(o.workspaceOps.pending, id)
	if message != "" {
		p.result <- errors.New(message)
	} else {
		p.result <- nil
	}
}

// InvokeWorkspace executes only a currently published public service action.
// Cancellation stops waiting; a delivered adopt/owner action is never replayed.
func (o *Owner) InvokeWorkspace(ctx context.Context, s Snapshot, op, id, path, owner string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o.mu.Lock()
	page := o.page
	valid := !o.closed && !o.incompatible && o.state.Connected && s.Subject != nil && o.state.Subject != nil && o.state.Subject.equal(*s.Subject) && o.selected.equal(*s.Subject) && page != nil && page.id == s.Page && o.state.Revision == s.Revision
	found := false
	if op == "directory" {
		for _, a := range o.state.DirectoryActions {
			if a.Handle == id {
				found = true
				break
			}
		}
	} else {
		for _, r := range o.state.WorkspaceNavigation {
			if r.SessionID == id && r.Owner == owner {
				found = op == "open" || op == "rename" && r.CanRename || op == "archive" && r.CanArchive || op == "restore" && r.CanRestore
				break
			}
		}
	}
	if !valid || !found {
		o.mu.Unlock()
		return errors.New("plugin UI: workspace action is no longer available")
	}
	o.workspaceOps.next++
	request := strconv.FormatUint(o.workspaceOps.next, 10)
	pending := &workspaceOperation{page: page, result: make(chan error, 1)}
	if o.workspaceOps.pending == nil {
		o.workspaceOps.pending = map[string]*workspaceOperation{}
	}
	o.workspaceOps.pending[request] = pending
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		if o.workspaceOps.pending[request] == pending {
			delete(o.workspaceOps.pending, request)
		}
		o.mu.Unlock()
	}()
	if err := o.sendToPage(page, map[string]any{"type": "workspaceAction", "request": request, "revision": s.Revision, "subject": s.Subject, "op": op, "id": id, "path": path, "owner": owner}); err != nil {
		return err
	}
	select {
	case err := <-pending.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (o *Owner) SetNavigationContext(id string) {
	if o == nil {
		return
	}
	o.mu.Lock()
	if o.navigationSession == id {
		o.mu.Unlock()
		return
	}
	o.navigationSession = id
	page := o.page
	o.mu.Unlock()
	if page != nil {
		_ = o.sendToPage(page, map[string]any{"type": "navigationContext", "sessionId": id})
	}
}

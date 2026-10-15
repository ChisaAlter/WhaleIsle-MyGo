// Package mcpconfig owns the managed MCP settings edit and mutation lifecycle.
package mcpconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"whaleisle/internal/dsh/api"
)

type Caller interface {
	Call(context.Context, string, any, any) error
}
type Snapshot struct {
	Import                                        ImportState
	Form                                          *FormDraft
	FormErrors                                    map[string]string
	Query, EnabledFilter, ExpandedID              string
	Tests                                         map[string]TestDiagnostic
	Rows                                          []api.McpServerSnapshotServersItem
	Ready, Busy                                   bool
	Draft, EditingID, DeletingID, Err, LastAction string
}
type Controller struct {
	importSequence uint64
	mu             sync.Mutex
	state          Snapshot
	revision       uint64
}

func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.state
	s.Import = cloneImport(s.Import)
	if c.state.Form != nil {
		form := *c.state.Form
		s.Form = &form
	}
	s.FormErrors = make(map[string]string, len(c.state.FormErrors))
	for key, message := range c.state.FormErrors {
		s.FormErrors[key] = message
	}
	s.Tests = cloneDiagnostics(s.Tests)
	s.Rows = append([]api.McpServerSnapshotServersItem(nil), s.Rows...)
	for i := range s.Rows {
		s.Rows[i].Spec = append(json.RawMessage(nil), s.Rows[i].Spec...)
		if connection := s.Rows[i].Connection; connection != nil {
			copy := *connection
			copy.Tools = append([]string(nil), connection.Tools...)
			if connection.LastError != nil {
				message := *connection.LastError
				copy.LastError = &message
			}
			s.Rows[i].Connection = &copy
		}
	}
	return s
}
func (c *Controller) writable(id string) bool {
	for _, r := range c.state.Rows {
		if r.Id == id {
			return r.Writable
		}
	}
	return false
}
func (c *Controller) Edit(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy || !c.writable(id) {
		return
	}
	for _, r := range c.state.Rows {
		if r.Id == id {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, r.Spec, "", "  "); err != nil {
				c.state.Err = "Invalid server configuration"
				return
			}
			c.cancelImportDraft()
			c.state.Form = nil
			c.state.FormErrors = nil
			c.state.Draft = pretty.String()
			c.state.EditingID = id
			c.state.Err = ""
			c.revision++
			return
		}
	}
}
func (c *Controller) SetDraft(text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Draft != text {
		c.state.Form = nil
		c.state.FormErrors = nil
		c.state.Draft = text
		c.revision++
	}
}
func (c *Controller) CancelEdit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy {
		return
	}
	c.cancelImportDraft()
	c.state.Form = nil
	c.state.FormErrors = nil
	c.state.Draft = ""
	c.state.EditingID = ""
	c.state.Err = ""
	c.revision++
}
func (c *Controller) RequestDelete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Busy && c.writable(id) {
		c.state.DeletingID = id
		c.state.Err = ""
	}
}
func (c *Controller) CancelDelete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Busy {
		c.state.DeletingID = ""
		c.state.Err = ""
	}
}
func (c *Controller) begin(client Caller) bool {
	if c.state.Busy {
		return false
	}
	if client == nil {
		c.state.Err = "MCP host is not connected"
		return false
	}
	c.state.Busy = true
	c.state.Err = ""
	c.state.LastAction = ""
	return true
}
func readRows(ctx context.Context, client Caller) (api.McpServerSnapshot, error) {
	var snap api.McpServerSnapshot
	err := client.Call(ctx, "mcpServers/list", nil, &snap)
	return snap, err
}
func (c *Controller) Refresh(ctx context.Context, client Caller) {
	c.mu.Lock()
	if !c.begin(client) {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	snap, err := readRows(ctx, client)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Busy = false
	if err != nil {
		c.state.Err = err.Error()
		return
	}
	c.state.Rows = snap.Servers
	c.reconcileDiagnostics()
	c.state.Ready = true
}
func validateDraft(text, editingID string) (json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &value); err != nil || value == nil {
		return nil, errors.New("Enter a server JSON object")
	}
	var id, transport, endpoint string
	json.Unmarshal(value["id"], &id)
	json.Unmarshal(value["transport"], &transport)
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("Server id is required")
	}
	if editingID != "" && id != editingID {
		return nil, errors.New("Keep the existing server id when editing")
	}
	switch transport {
	case "stdio":
		json.Unmarshal(value["command"], &endpoint)
	case "streamable-http":
		json.Unmarshal(value["url"], &endpoint)
	default:
		return nil, errors.New("Transport must be stdio or streamable-http")
	}
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("Command or URL is required")
	}
	return json.RawMessage(text), nil
}
func (c *Controller) Save(ctx context.Context, client Caller) {
	snap := c.Snapshot()
	c.SaveExpected(ctx, client, snap.Draft, snap.EditingID)
}

// SaveExpected rejects a queued click after the user has selected another draft.
func (c *Controller) SaveExpected(ctx context.Context, client Caller, draft, editingID string) {
	c.saveExpectedVersion(ctx, client, draft, editingID, nil)
}
func (c *Controller) saveExpectedVersion(ctx context.Context, client Caller, draft, editingID string, expectedRevision *uint64) {
	c.mu.Lock()
	if expectedRevision != nil && c.revision != *expectedRevision {
		c.mu.Unlock()
		return
	}
	if c.state.Draft != draft || c.state.EditingID != editingID {
		c.mu.Unlock()
		return
	}
	if c.state.Busy {
		c.mu.Unlock()
		return
	}
	spec, err := validateDraft(c.state.Draft, c.state.EditingID)
	if err == nil {
		var identity struct {
			ID string `json:"id"`
		}
		json.Unmarshal(spec, &identity)
		for _, r := range c.state.Rows {
			if r.Id == identity.ID && !r.Writable {
				err = errors.New("Server is read-only")
			}
		}
		if c.state.EditingID != "" && !c.writable(c.state.EditingID) {
			err = errors.New("Server is no longer writable")
		}
	}
	if err != nil {
		c.state.Err = err.Error()
		c.mu.Unlock()
		return
	}
	if !c.begin(client) {
		c.mu.Unlock()
		return
	}
	revision := c.revision
	c.mu.Unlock()
	var out api.UpsertResult
	err = client.Call(ctx, "mcpServers/upsert", map[string]any{"request": api.McpServerUpsertRequest{Spec: spec}}, &out)
	c.finish(ctx, client, err, "Saved", func() {
		if c.revision == revision {
			c.finishImportSave(spec)
			c.state.Form = nil
			c.state.FormErrors = nil
			c.state.Draft = ""
			c.state.EditingID = ""
		}
	})
}

// finish keeps mutation failures visible and distinguishes a successful write
// followed by a failed reload, so retrying does not silently duplicate a write.
func (c *Controller) finish(ctx context.Context, client Caller, err error, action string, success func()) {
	if err != nil {
		c.mu.Lock()
		c.state.Busy = false
		c.state.Err = err.Error()
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	if success != nil {
		success()
	}
	c.state.LastAction = action
	c.mu.Unlock()
	snap, refreshErr := readRows(ctx, client)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Busy = false
	if refreshErr != nil {
		c.state.Err = action + " completed, but refresh failed: " + refreshErr.Error()
		return
	}
	c.state.Rows = snap.Servers
	c.reconcileDiagnostics()
	c.state.Ready = true
}
func (c *Controller) Delete(ctx context.Context, client Caller) {
	c.DeleteExpected(ctx, client, c.Snapshot().DeletingID)
}

// DeleteExpected binds confirmation to the displayed target, even if queued.
func (c *Controller) DeleteExpected(ctx context.Context, client Caller, id string) {
	c.mu.Lock()
	if id != c.state.DeletingID {
		c.mu.Unlock()
		return
	}
	if id == "" || !c.writable(id) || !c.begin(client) {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	var out api.DeleteResult
	err := client.Call(ctx, "mcpServers/delete", map[string]any{"request": api.McpServerIdRequest{Id: id}}, &out)
	c.finish(ctx, client, err, "Removed "+id, func() {
		c.state.DeletingID = ""
		if c.state.EditingID == id {
			c.state.Form = nil
			c.state.FormErrors = nil
			c.state.EditingID = ""
			c.state.Draft = ""
			c.revision++
		}
	})
}
func (c *Controller) SetEnabled(ctx context.Context, client Caller, id string, enabled bool) {
	c.rowAction(ctx, client, id, "setEnabled", api.McpServerEnableRequest{Id: id, Enabled: enabled})
}
func (c *Controller) RowAction(ctx context.Context, client Caller, id, action string) {
	if action != "retry" && action != "authorize" && action != "test" {
		return
	}
	c.rowAction(ctx, client, id, action, api.McpServerIdRequest{Id: id})
}
func (c *Controller) rowAction(ctx context.Context, client Caller, id, action string, request any) {
	c.mu.Lock()
	exists := false
	for _, r := range c.state.Rows {
		if r.Id == id {
			exists = true
		}
	}
	if !exists || (action != "test" && !c.writable(id)) || !c.begin(client) {
		c.mu.Unlock()
		return
	}
	if action == "test" {
		delete(c.state.Tests, id)
	}
	c.mu.Unlock()
	var result json.RawMessage
	err := client.Call(ctx, "mcpServers/"+action, map[string]any{"request": request}, &result)
	message := action + ": " + id
	if action == "test" {
		c.mu.Lock()
		err = c.recordTest(id, result, err)
		c.mu.Unlock()
	}
	if err == nil && action == "test" {
		var test api.McpServerTestResult
		if json.Unmarshal(result, &test) == nil && test.Ok {
			message = fmt.Sprintf("Test OK: %s · %d tools", id, int(test.ToolCount))
		}
	}
	c.finish(ctx, client, err, message, nil)
}

// Package credentialrefs implements write-only native provider/Web Search keys.
// Secret literals live only in an explicitly edited in-memory draft and a set
// request. Neither raw RPC errors nor response bodies enter status messages.
package credentialrefs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"unicode"
)

type Snapshot struct {
	Rows                                 []Row
	SelectedID, Draft, Err, LastAction   string
	Ticket                               uint64
	Connected, Ready, Busy, ClearPending bool
}
type Controller struct {
	mu          sync.Mutex
	call        Caller
	hostContext context.Context
	cancel      context.CancelFunc
	state       Snapshot
}

// Attach invalidates pending requests and secrets whenever a Host is replaced.
// The desktop calls this at its existing attach/detach boundaries, not per frame.
func (c *Controller) Attach(call Caller) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.hostContext, c.cancel = context.WithCancel(context.Background())
	c.call = call
	c.state = Snapshot{Ticket: c.state.Ticket + 1, Connected: call != nil}
}
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.state
	s.Rows = append([]Row(nil), s.Rows...)
	for i := range s.Rows {
		s.Rows[i].Path = append([]string(nil), s.Rows[i].Path...)
	}
	return s
}
func (c *Controller) Select(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy || !c.state.Ready {
		return
	}
	for _, r := range c.state.Rows {
		if r.ID == id {
			c.state.SelectedID = id
			c.state.Ticket++
			c.state.Draft = ""
			c.state.ClearPending = false
			c.state.Err = ""
			c.state.LastAction = ""
			return
		}
	}
}
func (c *Controller) Edit(ticket uint64, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy {
		return
	}
	c.state.Draft = value
	c.state.ClearPending = false
	c.state.Err = ""
	c.state.LastAction = ""
}
func (c *Controller) StageClear(ticket uint64, clear bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy {
		return
	}
	c.state.ClearPending = clear
	c.state.Draft = ""
	c.state.Err = ""
	c.state.LastAction = ""
}
func (c *Controller) request(ctx context.Context, ticket uint64, call Caller) request {
	return func(method string, args, out any) error {
		c.mu.Lock()
		current := c.state.Ticket == ticket && c.call != nil
		c.mu.Unlock()
		if !current {
			return context.Canceled
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return call.Call(ctx, method, args, out)
	}
}
func (c *Controller) workContext(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(c.hostContext)
	stop := context.AfterFunc(parent, cancel)
	return ctx, func() { stop(); cancel() }
}
func (c *Controller) Refresh(parent context.Context) {
	c.mu.Lock()
	if c.call == nil || c.state.Busy {
		c.mu.Unlock()
		return
	}
	c.state.Ticket++
	ticket := c.state.Ticket
	call := c.call
	ctx, done := c.workContext(parent)
	c.state.Busy = true
	c.state.Ready = false
	c.state.Draft = ""
	c.state.ClearPending = false
	c.state.Err = ""
	c.state.LastAction = ""
	c.mu.Unlock()
	defer done()
	rows, err := load(c.request(ctx, ticket, call))
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket {
		return
	}
	c.state.Busy = false
	if err != nil {
		c.state.Rows = nil
		c.state.Err = err.Error()
		return
	}
	c.state.Rows = rows
	c.state.Ready = true
	found := false
	for _, r := range rows {
		if r.ID == c.state.SelectedID {
			found = true
		}
	}
	if !found {
		c.state.SelectedID = ""
	}
}
func selected(rows []Row, id string) (Row, bool) {
	for _, r := range rows {
		if r.ID == id {
			return r, true
		}
	}
	return Row{}, false
}

// Save checks the view's ticket, Host metadata, target reference and settings
// revision before a mutation. A blank input is always "keep existing".
func (c *Controller) Save(parent context.Context, ticket uint64) {
	c.mu.Lock()
	row, ok := selected(c.state.Rows, c.state.SelectedID)
	value := strings.TrimSpace(c.state.Draft)
	clear := c.state.ClearPending
	if c.call == nil || c.state.Busy || !c.state.Ready || c.state.Ticket != ticket || !ok || (!clear && value == "") {
		c.mu.Unlock()
		return
	}
	if !row.Known || !row.Writable {
		c.state.Err = "This credential is read-only or its writable state is unknown. Refresh before editing."
		c.mu.Unlock()
		return
	}
	if !clear && strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		c.state.Err = "API keys cannot contain internal whitespace."
		c.mu.Unlock()
		return
	}
	c.state.Busy = true
	c.state.Err = ""
	c.state.LastAction = ""
	call := c.call
	ctx, done := c.workContext(parent)
	c.mu.Unlock()
	defer done()
	request := c.request(ctx, ticket, call)
	rows, err := load(request)
	fresh, exists := selected(rows, row.ID)
	if err == nil && (!exists || fresh.Ref != row.Ref || fresh.Revision != row.Revision || fresh.BindReference != row.BindReference) {
		err = errors.New("The settings revision or credential reference changed. Refresh and re-enter the key before saving.")
	}
	if err == nil && (!fresh.Known || !fresh.Writable) {
		err = errors.New("The Host reports this credential as read-only. Change its owning source before editing.")
	}
	if err == nil && !clear && fresh.BindReference {
		var written namespace
		path := append(append([]string(nil), fresh.Path...), "apiKeyEnv")
		if request("settings/mutate", map[string]any{"ns": fresh.Namespace, "expectedRevision": fresh.Revision, "ops": []map[string]any{{"op": "set", "path": path, "value": fresh.Ref}}}, &written) != nil {
			err = errors.New("The provider reference could not be saved. Refresh if another settings editor changed it.")
		} else if written.NS != fresh.Namespace || profileRef(written.Value, fresh.Path) != fresh.Ref {
			err = errors.New("The provider reference write could not be verified. Refresh before retrying.")
		} else {
			// A successful settings write must advance the retry baseline even if the
			// next credential call fails. Never resubmit a stale revision on retry.
			fresh.Revision = written.Revision
			fresh.BindReference = false
			for i := range rows {
				if rows[i].ID == fresh.ID {
					rows[i] = fresh
				}
			}
			c.mu.Lock()
			if c.state.Ticket == ticket {
				c.state.Rows = rows
			}
			c.mu.Unlock()
		}
	}
	wrote := false
	if err == nil {
		method := "credentials/set"
		args := map[string]any{"ref": fresh.Ref, "value": value}
		if clear {
			method = "credentials/unset"
			args = map[string]any{"ref": fresh.Ref}
		}
		if request(method, args, nil) != nil {
			err = errors.New("The credential write was refused or failed. Check Host availability and whether this reference is writable, then retry.")
		} else {
			wrote = true
		}
	}
	if wrote {
		// Discard the submitted literal even if a subsequent read fails. Retrying an
		// uncertain read must not silently repeat a successfully acknowledged write.
		c.mu.Lock()
		if c.state.Ticket == ticket {
			c.state.Draft = ""
			c.state.ClearPending = false
		}
		c.mu.Unlock()
		rows, err = load(request)
		if err != nil {
			err = errors.New("The credential write was accepted, but refresh failed. Refresh to verify its state.")
		} else if !clear {
			verified, found := selected(rows, fresh.ID)
			if !found || verified.Ref != fresh.Ref || !verified.Known || !verified.Configured {
				err = errors.New("The credential write was accepted, but the Host has not confirmed it as configured. Refresh to verify its state.")
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket {
		return
	}
	c.state.Busy = false
	if err != nil {
		c.state.Err = err.Error()
		if wrote {
			c.state.Ready = false
		}
		return
	}
	c.state.Rows = rows
	c.state.Draft = ""
	c.state.ClearPending = false
	if clear {
		c.state.LastAction = "Local credential cleared. An inherited credential may still be configured."
	} else {
		c.state.LastAction = "Credential saved and confirmed by the Host."
	}
}

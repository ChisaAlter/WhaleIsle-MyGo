package jobflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Stream interface {
	Recv(context.Context) (json.RawMessage, error)
	Cancel()
}
type Transport interface {
	Open(context.Context, string, map[string]any) (Stream, error)
	Kill(context.Context, map[string]any) (string, error)
	Retryable(error) bool
}
type KillState struct{ JobID, Phase, Error string }
type Snapshot struct {
	Generation                    uint64
	SessionID                     string
	Ready, Mounted, Open, Loading bool
	SettledOpen                   *bool
	Error                         string
	Rows                          []Job
	SelectedID                    string
	Observed                      Observation
	Kill                          KillState
}
type Controller struct {
	mu                sync.Mutex
	notify            func()
	transport         Transport
	view              Snapshot
	ctx               context.Context
	cancel            context.CancelFunc
	rowsCancel        context.CancelFunc
	rowsEpoch         uint64
	observationCancel context.CancelFunc
	observationEpoch  uint64
	cleared           map[string]bool
	killEpoch         uint64
}

func New(notify func()) *Controller { return &Controller{notify: notify} }

// Bind resets all view and request state. The owner calls this only when its
// Host, mux, session, or session generation changes; it does not notify under
// the owner's lock. Cancellation touches only our logical streams.
func (c *Controller) Bind(t Transport, sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.stopObservationLocked()
	c.rowsEpoch++
	c.killEpoch++
	g := c.view.Generation + 1
	c.view = Snapshot{Generation: g, SessionID: sessionID, Ready: t != nil && sessionID != ""}
	c.transport = t
	c.cancel = nil
	c.rowsCancel = nil
	c.cleared = nil
}
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.view
	if v.SettledOpen != nil {
		value := *v.SettledOpen
		v.SettledOpen = &value
	}
	v.Rows = nil
	for _, j := range c.view.Rows {
		if !c.cleared[j.ID] || j.Live() {
			j.Output.SpillPaths = append([]string(nil), j.Output.SpillPaths...)
			v.Rows = append(v.Rows, j)
		}
	}
	order(v.Rows)
	return v
}
func (c *Controller) changed() {
	if c.notify != nil {
		c.notify()
	}
}
func (c *Controller) Open(g uint64) {
	c.mu.Lock()
	if c.view.Generation != g || !c.view.Ready || c.view.Open {
		c.mu.Unlock()
		return
	}
	c.view.Open = true
	if !c.view.Mounted {
		c.mountLocked()
	}
	c.mu.Unlock()
	c.changed()
}

// Mount follows the visible session header, independently of its popup.
func (c *Controller) Mount() {
	c.mu.Lock()
	if !c.view.Ready || c.view.Mounted {
		c.mu.Unlock()
		return
	}
	c.mountLocked()
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) mountLocked() {
	c.view.Mounted = true
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.startRowsLocked()
}
func (c *Controller) Unmount() {
	c.mu.Lock()
	if !c.view.Mounted {
		c.mu.Unlock()
		return
	}
	if c.cancel != nil {
		c.cancel()
	}
	c.stopObservationLocked()
	c.rowsEpoch++
	c.killEpoch++
	c.view.Generation++
	c.view.Mounted, c.view.Open, c.view.Loading = false, false, false
	c.view.Rows, c.view.SettledOpen = nil, nil
	c.view.Error, c.view.Kill = "", KillState{}
	c.ctx, c.cancel, c.rowsCancel, c.cleared = nil, nil, nil, nil
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) Close(g uint64) {
	c.mu.Lock()
	if c.view.Generation != g {
		c.mu.Unlock()
		return
	}
	c.stopObservationLocked()
	c.view.Open = false
	c.view.Generation++
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) Retry(g uint64) {
	c.mu.Lock()
	if c.view.Generation != g || !c.view.Mounted {
		c.mu.Unlock()
		return
	}
	c.startRowsLocked()
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) startRowsLocked() {
	if c.rowsCancel != nil {
		c.rowsCancel()
	}
	c.rowsEpoch++
	ctx, cancel := context.WithCancel(c.ctx)
	c.rowsCancel = cancel
	c.view.Loading = true
	c.view.Error = ""
	go c.runRows(ctx, c.transport, c.view.SessionID, c.rowsEpoch)
}
func (c *Controller) currentLocked(ctx context.Context, g uint64) bool {
	return ctx.Err() == nil && c.view.Open && c.view.Generation == g
}
func (c *Controller) runRows(ctx context.Context, t Transport, sessionID string, e uint64) {
	accepted := false
	for {
		stream, err := t.Open(ctx, "job/list", map[string]any{"sessionId": sessionID})
		if err == nil {
			for {
				var raw json.RawMessage
				raw, err = stream.Recv(ctx)
				if err != nil {
					break
				}
				var f struct {
					Type string `json:"type"`
					Jobs []Job  `json:"jobs"`
				}
				if err = json.Unmarshal(raw, &f); err != nil {
					break
				}
				if f.Type != "rows" {
					err = fmt.Errorf("unexpected jobs roster frame %q", f.Type)
					break
				}
				c.mu.Lock()
				if ctx.Err() != nil || !c.view.Mounted || c.rowsEpoch != e {
					c.mu.Unlock()
					stream.Cancel()
					return
				}
				c.view.Rows = f.Jobs
				c.view.Loading = false
				c.view.Error = ""
				accepted = true
				if c.view.SelectedID != "" {
					selected := c.rowLocked(c.view.SelectedID)
					if selected == nil || !selected.Observable() {
						c.stopObservationLocked()
					}
				}
				kill := c.rowLocked(c.view.Kill.JobID)
				if kill == nil || kill.Status != "running" {
					c.view.Kill = KillState{}
					c.killEpoch++
				}
				for _, j := range f.Jobs {
					if j.Live() {
						delete(c.cleared, j.ID)
					}
				}
				c.closeEmptyLocked()
				c.mu.Unlock()
				c.changed()
			}
			stream.Cancel()
		}
		c.mu.Lock()
		if ctx.Err() != nil || !c.view.Mounted || c.rowsEpoch != e {
			c.mu.Unlock()
			return
		}
		retry := accepted && t.Retryable(err)
		c.view.Loading = retry
		c.view.Error = err.Error()
		c.view.Kill = KillState{}
		c.killEpoch++
		c.stopObservationLocked()
		c.mu.Unlock()
		c.changed()
		if !retry || !pause(ctx) {
			return
		}
	}
}
func pause(ctx context.Context) bool {
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (c *Controller) rowLocked(id string) *Job {
	for i := range c.view.Rows {
		if c.view.Rows[i].ID == id {
			return &c.view.Rows[i]
		}
	}
	return nil
}
func (c *Controller) hasRowLocked(id string) bool { return c.rowLocked(id) != nil }
func (c *Controller) stopObservationLocked() {
	if c.observationCancel != nil {
		c.observationCancel()
	}
	c.observationEpoch++
	c.observationCancel = nil
	c.view.SelectedID = ""
	c.view.Observed = Observation{}
}
func (c *Controller) Toggle(g uint64, id string) {
	c.mu.Lock()
	if c.view.Generation != g || !c.view.Open {
		c.mu.Unlock()
		return
	}
	row := c.rowLocked(id)
	if row == nil || !row.Observable() {
		c.mu.Unlock()
		return
	}
	if c.view.SelectedID == id {
		c.stopObservationLocked()
		c.mu.Unlock()
		c.changed()
		return
	}
	c.stopObservationLocked()
	c.view.SelectedID = id
	c.view.Observed = Observation{JobID: id, Streaming: true}
	ctx, cancel := context.WithCancel(c.ctx)
	c.observationCancel = cancel
	go c.runObservation(ctx, c.transport, row.Owner, id, g, c.observationEpoch)
	c.mu.Unlock()
	c.changed()
}

// RetryOutput preserves the accepted byte cursor and tail after an observation
// failure; closing/collapsing the panel, by contrast, releases its output state.
func (c *Controller) RetryOutput(g uint64) {
	c.mu.Lock()
	row := c.rowLocked(c.view.SelectedID)
	if c.view.Generation != g || !c.view.Open || row == nil {
		c.mu.Unlock()
		return
	}
	if c.observationCancel != nil {
		c.observationCancel()
	}
	c.observationEpoch++
	c.view.Observed.Error = ""
	c.view.Observed.Streaming = true
	ctx, cancel := context.WithCancel(c.ctx)
	c.observationCancel = cancel
	go c.runObservation(ctx, c.transport, row.Owner, c.view.SelectedID, g, c.observationEpoch)
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) runObservation(ctx context.Context, t Transport, owner, id string, g, e uint64) {
	accepted := false
	for {
		c.mu.Lock()
		if !c.currentLocked(ctx, g) || c.observationEpoch != e {
			c.mu.Unlock()
			return
		}
		request := map[string]any{"jobId": id}
		if owner != "" {
			request["sessionId"] = owner
		}
		if c.view.Observed.HasCursor {
			request["from"] = c.view.Observed.Cursor
		}
		c.mu.Unlock()
		stream, err := t.Open(ctx, "job/follow", request)
		terminal := false
		if err == nil {
			for {
				var raw json.RawMessage
				raw, err = stream.Recv(ctx)
				if err != nil {
					break
				}
				var f Frame
				if err = json.Unmarshal(raw, &f); err != nil {
					break
				}
				c.mu.Lock()
				if !c.currentLocked(ctx, g) || c.observationEpoch != e {
					c.mu.Unlock()
					stream.Cancel()
					return
				}
				err = c.view.Observed.Apply(f)
				if err == nil && f.Type == "opened" {
					accepted = true
				}
				terminal = err == nil && f.Type == "status"
				c.mu.Unlock()
				c.changed()
				if err != nil || terminal {
					break
				}
			}
			stream.Cancel()
		}
		if terminal {
			return
		}
		c.mu.Lock()
		if !c.currentLocked(ctx, g) || c.observationEpoch != e {
			c.mu.Unlock()
			return
		}
		retry := accepted && t.Retryable(err)
		c.view.Observed.Error = err.Error()
		c.view.Observed.Streaming = retry
		c.mu.Unlock()
		c.changed()
		if !retry || !pause(ctx) {
			return
		}
	}
}
func (c *Controller) ClearSettled(g uint64) {
	c.mu.Lock()
	if c.view.Generation != g || !c.view.Open {
		c.mu.Unlock()
		return
	}
	if c.cleared == nil {
		c.cleared = map[string]bool{}
	}
	for _, j := range c.view.Rows {
		if !j.Live() {
			c.cleared[j.ID] = true
			if c.view.SelectedID == j.ID {
				c.stopObservationLocked()
			}
		}
	}
	c.closeEmptyLocked()
	c.mu.Unlock()
	c.changed()
}

func (c *Controller) closeEmptyLocked() {
	for _, row := range c.view.Rows {
		if row.Live() || !c.cleared[row.ID] {
			return
		}
	}
	if c.view.Open {
		c.view.Open = false
		c.view.Generation++
		c.stopObservationLocked()
	}
}
func (c *Controller) ToggleSettled(g uint64) {
	c.mu.Lock()
	if c.view.Generation != g || !c.view.Open {
		c.mu.Unlock()
		return
	}
	open := true
	for _, row := range c.view.Rows {
		if row.Live() {
			open = false
			break
		}
	}
	if c.view.SettledOpen != nil {
		open = *c.view.SettledOpen
	}
	open = !open
	c.view.SettledOpen = &open
	c.mu.Unlock()
	c.changed()
}
func (c *Controller) ArmKill(g uint64, id string) {
	c.mu.Lock()
	j := c.rowLocked(id)
	if c.view.Generation != g || !c.view.Open || c.view.Loading || c.view.Error != "" || j == nil || j.Status != "running" || c.view.Kill.Phase == "pending" {
		c.mu.Unlock()
		return
	}
	c.killEpoch++
	e := c.killEpoch
	c.view.Kill = KillState{JobID: id, Phase: "armed"}
	c.mu.Unlock()
	c.changed()
	time.AfterFunc(3*time.Second, func() {
		c.mu.Lock()
		changed := c.killEpoch == e && c.view.Kill.Phase == "armed"
		if changed {
			c.view.Kill = KillState{}
		}
		c.mu.Unlock()
		if changed {
			c.changed()
		}
	})
}
func (c *Controller) CancelKill(g uint64) {
	c.mu.Lock()
	changed := c.view.Generation == g && c.view.Kill.Phase != "pending"
	if changed {
		c.killEpoch++
		c.view.Kill = KillState{}
	}
	c.mu.Unlock()
	if changed {
		c.changed()
	}
}
func (c *Controller) ConfirmKill(g uint64, id string) {
	c.mu.Lock()
	j := c.rowLocked(id)
	if c.view.Generation != g || !c.view.Open || c.view.Loading || c.view.Error != "" || j == nil || j.Status != "running" || c.view.Kill.JobID != id || c.view.Kill.Phase != "armed" {
		c.mu.Unlock()
		return
	}
	c.killEpoch++
	e := c.killEpoch
	c.view.Kill.Phase = "pending"
	t, sessionID := c.transport, c.view.SessionID
	ownerContext := c.ctx
	ctx, cancel := context.WithTimeout(ownerContext, 30*time.Second)
	c.mu.Unlock()
	c.changed()
	go func() {
		defer cancel()
		outcome, err := t.Kill(ctx, map[string]any{"sessionId": sessionID, "jobId": id})
		if err == nil && outcome != "requested" && outcome != "already-finished" {
			err = fmt.Errorf("unexpected job kill receipt %q", outcome)
		}
		c.mu.Lock()
		if ownerContext.Err() != nil || !c.view.Mounted || c.killEpoch != e {
			c.mu.Unlock()
			return
		}
		if err != nil {
			c.view.Kill.Phase = "failed"
			c.view.Kill.Error = err.Error()
		}
		c.mu.Unlock()
		c.changed()
		if err != nil {
			time.AfterFunc(4*time.Second, func() {
				c.mu.Lock()
				changed := ownerContext.Err() == nil && c.view.Mounted && c.killEpoch == e && c.view.Kill.JobID == id && c.view.Kill.Phase == "failed"
				if changed {
					c.view.Kill = KillState{}
				}
				c.mu.Unlock()
				if changed {
					c.changed()
				}
			})
		}
	}()
}

package providerdiscovery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Rows                                                        []Row
	SelectedID, Endpoint, Protocol, Key, Query, Err, LastAction string
	Models                                                      []map[string]any
	Candidates                                                  []Candidate
	Ticket                                                      uint64
	Connected, Ready, Busy, Dirty, Review                       bool
}
type Controller struct {
	mu              sync.Mutex
	call            Caller
	hostContext     context.Context
	cancel          context.CancelFunc
	probeCancel     func()
	visible, saving bool
	state           Snapshot
}

func (c *Controller) Attach(call Caller) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.hostContext, c.cancel = context.WithCancel(context.Background())
	c.call = call
	c.probeCancel = nil
	c.saving = false
	c.state = Snapshot{Connected: call != nil, Ticket: c.state.Ticket + 1}
}

// ObserveVisible is driven by the actual rendered page and window visibility.
// Hiding removes write-only secrets and pending confirmations without discarding
// non-secret model/endpoint edits. An already dispatched save keeps its verifier.
func (c *Controller) ObserveVisible(visible bool) {
	if !visible {
		c.Suspend()
		return
	}
	c.mu.Lock()
	c.visible = true
	c.mu.Unlock()
}
func (c *Controller) Suspend() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.visible && c.state.Key == "" && !c.state.Review && c.probeCancel == nil {
		return
	}
	c.visible = false
	c.state.Key, c.state.Query = "", ""
	c.state.Review = false
	c.state.Candidates = nil
	if c.saving {
		return
	} // Save already consumed its confirmation ticket.
	c.state.Ticket++
	if c.probeCancel != nil {
		c.probeCancel()
		c.probeCancel = nil
		c.state.LastAction = "Discovery was cancelled because the Models page was hidden. The temporary key was discarded."
	}
	c.state.Busy = false
}
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.state
	s.Rows = append([]Row(nil), s.Rows...)
	for i := range s.Rows {
		s.Rows[i].Path = append([]string(nil), s.Rows[i].Path...)
		s.Rows[i].Models = cloneModels(s.Rows[i].Models)
	}
	s.Models = cloneModels(s.Models)
	s.Candidates = append([]Candidate(nil), s.Candidates...)
	for i := range s.Candidates {
		if s.Candidates[i].InputModalities != nil {
			s.Candidates[i].InputModalities = append([]string{}, s.Candidates[i].InputModalities...)
		}
	}
	return s
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
	ctx, cancel := context.WithTimeout(c.hostContext, 40*time.Second)
	stop := context.AfterFunc(parent, cancel)
	return ctx, func() { stop(); cancel() }
}
func (c *Controller) resetDraft(row Row) {
	c.state.SelectedID = row.ID
	c.state.Endpoint, c.state.Protocol = row.Endpoint, row.Protocol
	c.state.Models = cloneModels(row.Models)
	c.state.Key, c.state.Query, c.state.Err, c.state.LastAction = "", "", "", ""
	c.state.Candidates = nil
	c.state.Dirty, c.state.Review = false, false
}
func (c *Controller) Refresh(parent context.Context) {
	c.mu.Lock()
	if c.call == nil || c.state.Busy || c.state.Dirty {
		c.mu.Unlock()
		return
	}
	c.state.Ticket++
	ticket, call := c.state.Ticket, c.call
	ctx, done := c.workContext(parent)
	c.state.Busy, c.state.Ready = true, false
	c.state.Key, c.state.Err, c.state.LastAction = "", "", ""
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
		c.state.Err = err.Error()
		return
	}
	oldID := c.state.SelectedID
	c.state.Rows, c.state.Ready = rows, true
	if row, ok := findRow(rows, oldID); ok {
		c.resetDraft(row)
	} else {
		c.resetDraft(Row{})
	}
}
func (c *Controller) Select(ticket uint64, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy || c.state.Dirty || !c.state.Ready {
		return
	}
	if row, ok := findRow(c.state.Rows, id); ok {
		c.state.Ticket++
		c.resetDraft(row)
	}
}
func (c *Controller) Edit(ticket uint64, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy || !c.state.Ready {
		return
	}
	row, ok := findRow(c.state.Rows, c.state.SelectedID)
	if !ok {
		return
	}
	switch field {
	case "endpoint":
		c.state.Endpoint = value
		c.state.Candidates = nil
	case "protocol":
		c.state.Protocol = value
		c.state.Candidates = nil
	case "key":
		c.state.Key = value
		c.state.Candidates = nil
	case "query":
		c.state.Query = value
		return
	default:
		return
	}
	c.state.Ticket++
	c.state.Review = false
	c.state.Err, c.state.LastAction = "", ""
	c.state.Dirty = c.state.Endpoint != row.Endpoint || c.state.Protocol != row.Protocol || !reflect.DeepEqual(c.state.Models, row.Models)
}
func (c *Controller) Discard(ticket uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy {
		return
	}
	if row, ok := findRow(c.state.Rows, c.state.SelectedID); ok {
		c.state.Ticket++
		c.resetDraft(row)
	}
}
func (c *Controller) Discover(parent context.Context, ticket uint64) {
	c.mu.Lock()
	row, ok := findRow(c.state.Rows, c.state.SelectedID)
	if c.call == nil || c.state.Ticket != ticket || c.state.Busy || !c.state.Ready || !ok {
		c.mu.Unlock()
		return
	}
	c.state.Ticket++
	ticket = c.state.Ticket
	endpoint, protocol, key := c.state.Endpoint, c.state.Protocol, strings.TrimSpace(c.state.Key)
	ctx, done := c.workContext(parent)
	c.state.Key, c.state.Err, c.state.LastAction = "", "", ""
	c.probeCancel = done
	c.state.Candidates = nil
	c.state.Busy, c.state.Review = true, false
	call := c.call
	c.mu.Unlock()
	defer done()
	candidates, err := discover(c.request(ctx, ticket, call), row, endpoint, protocol, key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || ctx.Err() != nil {
		if c.state.Ticket == ticket {
			c.probeCancel = nil
			c.state.Busy = false
			c.state.Err = "Discovery was cancelled. The temporary key was discarded."
		}
		return
	}
	c.state.Busy = false
	c.probeCancel = nil
	if err != nil {
		c.state.Err = err.Error()
		return
	}
	known := map[string]bool{}
	for _, model := range c.state.Models {
		known[modelID(model)] = true
	}
	for _, candidate := range candidates {
		// A provider response cannot turn a reflected probe secret into a saved model.
		if key != "" && (strings.Contains(candidate.ID, strings.TrimSpace(key)) || strings.Contains(candidate.Name, strings.TrimSpace(key))) {
			continue
		}
		candidate.Picked = !known[candidate.ID]
		c.state.Candidates = append(c.state.Candidates, candidate)
	}
	c.state.Query = ""
	if len(c.state.Candidates) == 0 {
		c.state.LastAction = "No models were returned. No settings were changed."
	} else {
		c.state.LastAction = fmt.Sprintf("%s: %d candidates. Select models, add them to the draft, then review and save.", row.Name, len(c.state.Candidates))
	}
}

// ProbeEditor reuses the Host adapter probe for the current native form. Its
// result belongs to that form; the standalone discovery Models/Dirty/Review
// state and its Save entry point remain untouched.
func (c *Controller) ProbeEditor(parent context.Context, target EditorProbe) ([]Candidate, error) {
	c.mu.Lock()
	if c.call == nil || c.state.Busy {
		c.mu.Unlock()
		return nil, errors.New("Model discovery is unavailable while another operation is running.")
	}
	c.state.Ticket++
	ticket, call := c.state.Ticket, c.call
	ctx, done := c.workContext(parent)
	c.probeCancel, c.state.Busy = done, true
	c.mu.Unlock()
	defer done()
	candidates, err := discover(c.request(ctx, ticket, call), Row{Namespace: target.Namespace, Provider: target.Provider}, target.Endpoint, target.Protocol, target.Key)
	if err == nil {
		candidates = enrichDiscoveredCandidates(ctx, candidates)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || ctx.Err() != nil {
		if c.state.Ticket == ticket {
			c.probeCancel, c.state.Busy = nil, false
		}
		return nil, context.Canceled
	}
	c.probeCancel, c.state.Busy = nil, false
	if err != nil {
		return nil, err
	}
	filtered := candidates[:0]
	key := strings.TrimSpace(target.Key)
	for _, candidate := range candidates {
		if key != "" && (strings.Contains(candidate.ID, key) || strings.Contains(candidate.Name, key)) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered, nil
}

func (c *Controller) Pick(ticket uint64, id string, picked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy {
		return
	}
	for i := range c.state.Candidates {
		if c.state.Candidates[i].ID == id {
			c.state.Candidates[i].Picked = picked
		}
	}
}
func (c *Controller) Adopt(ticket uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket || c.state.Busy {
		return
	}
	row, ok := findRow(c.state.Rows, c.state.SelectedID)
	if !ok || !row.Writable || row.Blocked != "" {
		return
	}
	c.state.Models = mergeCandidates(c.state.Models, c.state.Candidates)
	c.state.Dirty = c.state.Endpoint != row.Endpoint || c.state.Protocol != row.Protocol || !reflect.DeepEqual(c.state.Models, row.Models)
	c.state.Candidates = nil
	c.state.Review = false
	c.state.Ticket++
	c.state.LastAction = "Selected models were added to the draft. Existing model fields were preserved. Review and save to apply."
}
func (c *Controller) Review(ticket uint64, review bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket == ticket && !c.state.Busy && c.state.Dirty {
		c.state.Review = review
		c.state.Key = ""
	}
}
func (c *Controller) Save(parent context.Context, ticket uint64) bool {
	c.mu.Lock()
	row, ok := findRow(c.state.Rows, c.state.SelectedID)
	if c.call == nil || c.state.Ticket != ticket || c.state.Busy || !c.state.Dirty || !c.state.Review || !ok || !row.Writable || row.Blocked != "" {
		c.mu.Unlock()
		return false
	}
	endpoint, protocol, models := strings.TrimSpace(c.state.Endpoint), strings.TrimSpace(c.state.Protocol), cloneModels(c.state.Models)
	if len(models) == 0 {
		c.state.Err = "Select at least one model before saving this discovery draft."
		c.mu.Unlock()
		return false
	}
	c.state.Ticket++
	ticket = c.state.Ticket
	call := c.call
	ctx, done := c.workContext(parent)
	c.saving = true
	c.state.Busy, c.state.Review = true, false
	c.state.Err, c.state.LastAction, c.state.Key = "", "", ""
	c.mu.Unlock()
	defer done()
	request := c.request(ctx, ticket, call)
	rows, err := load(request)
	fresh, exists := findRow(rows, row.ID)
	if err == nil && (!exists || fresh.Revision != row.Revision || !fresh.Writable || fresh.Blocked != "" || fresh.Ref != row.Ref) {
		err = errors.New("Provider settings changed or became read-only. Discard and refresh before saving; your draft was kept.")
	}
	var written namespace
	if err == nil {
		ops := []map[string]any{}
		if endpoint != row.Endpoint {
			ops = append(ops, map[string]any{"op": "set", "path": fieldPath(row, "baseURL"), "value": endpoint})
		}
		if protocol != row.Protocol {
			ops = append(ops, map[string]any{"op": "set", "path": fieldPath(row, "api"), "value": protocol})
		}
		if !reflect.DeepEqual(models, row.Models) {
			ops = append(ops, map[string]any{"op": "set", "path": fieldPath(row, "models"), "value": models})
		}
		if request("settings/mutate", map[string]any{"ns": row.Namespace, "expectedRevision": row.Revision, "ops": ops}, &written) != nil {
			err = errors.New("The settings write failed or conflicted. Your draft was kept; refresh after resolving a conflict.")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket != ticket {
		return false
	}
	c.saving = false
	c.state.Busy = false
	if err != nil {
		c.state.Err = err.Error()
		return false
	}
	profile := profileAt(written.Value, row.Path)
	// Acknowledgement alone is insufficient: compare the written fields and
	// advance the baseline only after the Host returned the committed view.
	encoded, _ := jsonModels(models)
	actual, _ := jsonModels(modelsAt(profile))
	if written.NS != row.Namespace || written.Revision <= row.Revision || encoded != actual || (endpoint != row.Endpoint && textOf(profile["baseURL"]) != endpoint) || ((protocol != row.Protocol) && textOf(profile["api"]) != protocol) {
		c.state.Ready = false
		c.state.Dirty = false
		c.state.Err = "The Host accepted the write, but its result could not be verified. Refresh before editing again."
		return false
	}
	row.Endpoint, row.Protocol, row.Revision, row.Models = endpoint, protocol, written.Revision, modelsAt(profile)
	row.ProtocolConfigured = protocol != ""
	for i := range c.state.Rows {
		if c.state.Rows[i].ID == row.ID {
			c.state.Rows[i] = row
		}
	}
	c.state.Ticket++
	c.resetDraft(row)
	c.state.LastAction = "Provider model settings were saved and verified. The temporary discovery key was not stored."
	return true
}

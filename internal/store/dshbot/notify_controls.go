package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type NotifyTicket struct {
	Scope                RoutineScope
	Generation, Sequence uint64
}
type NotifyConfirmation struct {
	Command      pluginrpc.DshbotNotifyCommand
	Sequence     uint64
	Acknowledged bool
}
type NotifySnapshot struct {
	Scope                                               RoutineScope
	Generation                                          uint64
	Loaded, Busy, Writable, Dirty, Uncertain, Candidate bool
	Draft                                               bool
	Status                                              pluginrpc.DshbotCapabilityStatus
	Review                                              *NotifyConfirmation
	Warnings                                            []string
	Error                                               *state.Fault
	Notice                                              string
}
type notifyAttempt struct {
	Command  pluginrpc.DshbotNotifyCommand
	Sequence uint64
	Sent     bool
}
type notifyEntry struct {
	state    NotifySnapshot
	attempt  *notifyAttempt
	loading  uint64
	hasDraft bool
	pub      state.Published[NotifySnapshot]
	dirty    bool
}

// Scope-local entries preserve sent requests across hidden/reopened frames. No
// old callback can dispatch, adopt or write status into a replacement account.
type NotifyControls struct {
	mu                   sync.Mutex
	active               RoutineScope
	generation, sequence uint64
	entries              map[RoutineScope]*notifyEntry
}

func (c *NotifyControls) entry(k RoutineScope) *notifyEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*notifyEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &notifyEntry{state: NotifySnapshot{Scope: k}}
	}
	c.entries[k].dirty = true
	return c.entries[k]
}
func (c *NotifyControls) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && c.active == k
}
func (c *NotifyControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == k {
		return
	}
	c.generation++
	c.active = k
	if k.HostKey != "" {
		e := c.entry(k)
		e.state.Loaded = false
		e.state.Review = nil
		e.state.Candidate = false
		e.loading = 0
		if e.attempt == nil || e.state.Uncertain {
			e.state.Busy = false
		}
	}
}

// Snapshot returns the published view for scope k; unchanged reads share
// the version and allocate nothing.
func (c *NotifyControls) Snapshot(k RoutineScope) *state.Version[NotifySnapshot] {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		e = c.entry(k)
	}
	if v := e.pub.Load(); v != nil && !e.dirty {
		return v
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotCapabilities(v.Status)
	v.Warnings = append([]string(nil), v.Warnings...)
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotNotifyCommand(r.Command)
		v.Review = &r
	}
	e.pub.Publish(v)
	e.dirty = false
	return e.pub.Load()
}
func (c *NotifyControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *NotifyControls) BeginLoad(k RoutineScope) (NotifyTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return NotifyTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return NotifyTicket{}, false
	}
	c.sequence++
	c.generation++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = nil
	return NotifyTicket{k, c.generation, c.sequence}, true
}
func (c *NotifyControls) LoadCurrent(t NotifyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *NotifyControls) FinishLoad(t NotifyTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotNotifyEnabled(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = state.FaultText(storelocale.T(storelocale.KeyStoreDshbotReadUnavailableDraftRetained2))
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = nil
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotNotifyApplied(e.attempt.Command, status) == nil {
			e.state.Candidate = true
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotMatchingProfileFoundInspectAndAdopt2)
		} else {
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownOutcomeRemainsLocked)
		}
		return true
	}
	if !e.hasDraft || !e.state.Dirty || e.state.Draft == value {
		e.state.Draft = value
	}
	e.hasDraft = true
	e.state.Dirty = e.state.Draft != value
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadBackReviewRemainingChanges)
	return true
}
func (c *NotifyControls) editable(k RoutineScope) *notifyEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if !e.state.Loaded || !e.state.Writable || e.state.Busy || e.attempt != nil || e.state.Status.WriteError != "" {
		return nil
	}
	return e
}
func (c *NotifyControls) SetDraft(k RoutineScope, v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return
	}
	before, err := pluginrpc.DshbotNotifyEnabled(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = v
	e.state.Dirty = v != before
	e.hasDraft = true
	c.generation++
}
func (c *NotifyControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	before, err := pluginrpc.DshbotNotifyEnabled(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = before
	e.state.Dirty = false
	e.state.Review = nil
	e.state.Error = nil
	c.generation++
}
func (c *NotifyControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Dirty {
		return
	}
	cmd := pluginrpc.DshbotNotifyCommand{Expected: pluginrpc.CloneDshbotCapabilities(e.state.Status), Enabled: e.state.Draft}
	if err := pluginrpc.ValidateDshbotNotifyCommand(cmd); err != nil {
		e.state.Error = state.FaultOf(err)
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &NotifyConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = nil
}
func (c *NotifyControls) Dismiss(k RoutineScope, r *NotifyConfirmation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && r != nil && e.state.Review != nil && r.Sequence == e.state.Review.Sequence {
		e.state.Review = nil
		c.generation++
	}
}
func (c *NotifyControls) Acknowledge(k RoutineScope, r *NotifyConfirmation, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && r != nil && e.state.Review != nil && r.Sequence == e.state.Review.Sequence {
		e.state.Review.Acknowledged = ack
		c.generation++
	}
}
func (c *NotifyControls) BeginSave(k RoutineScope, r *NotifyConfirmation) (NotifyTicket, pluginrpc.DshbotNotifyCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotNotifyCommand
	e := c.editable(k)
	if e == nil {
		return NotifyTicket{}, zero, false
	}
	if r == nil || e.state.Review == nil || e.state.Review.Sequence != r.Sequence || !e.state.Review.Acknowledged || !pluginrpc.SameDshbotCapabilities(e.state.Status, r.Command.Expected) || e.state.Draft != r.Command.Enabled {
		return NotifyTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotNotifyCommand(e.state.Review.Command)
	c.sequence++
	c.generation++
	e.attempt = &notifyAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = nil
	return NotifyTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *NotifyControls) current(t NotifyTicket) bool {
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *NotifyControls) Current(t NotifyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *NotifyControls) MarkSent(t NotifyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *NotifyControls) AbortUnsent(t NotifyTicket, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || e.attempt.Sent {
		return
	}
	e.attempt = nil
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotNotSentRefreshAndReview)
	if err != nil {
		e.state.Error = state.FaultOf(err)
	}
}
func (c *NotifyControls) FinishSave(t NotifyTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Warnings = append([]string(nil), out.Warnings...)
	if !out.Attempted {
		e.attempt = nil
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotNotSentRefresh)
		if err != nil {
			e.state.Error = state.FaultOf(err)
		}
		return
	}
	if err == nil {
		err = pluginrpc.DshbotNotifyApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = state.FaultOf(err)
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownReadBackOnly)
		return
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = e.attempt.Command.Enabled
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = nil
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotProfileSavedVerifyPresentationWarnin2)
	e.attempt = nil
}
func (c *NotifyControls) Adopt(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate || pluginrpc.DshbotNotifyApplied(e.attempt.Command, e.state.Status) != nil {
		return false
	}
	e.state.Draft = e.attempt.Command.Enabled
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Review = nil
	e.state.Error = nil
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotAdoptedCatalogPresentationRenameAnd)
	e.attempt = nil
	c.generation++
	return true
}

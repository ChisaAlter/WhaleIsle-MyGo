package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type SenderPolicyTicket struct {
	Scope                RoutineScope
	Generation, Sequence uint64
}
type SenderPolicyConfirmation struct {
	Command               pluginrpc.DshbotSenderPolicyCommand
	Sequence              uint64
	Acknowledged          bool
	ExpansionAcknowledged bool
}
type SenderPolicySnapshot struct {
	Scope                                               RoutineScope
	Generation                                          uint64
	Loaded, Busy, Writable, Dirty, Uncertain, Candidate bool
	Draft                                               pluginrpc.DshbotSenderPolicyValues
	Status                                              pluginrpc.DshbotSenderPolicyStatus
	Review                                              *SenderPolicyConfirmation
	Warnings                                            []string
	Error, Notice                                       string
}
type senderPolicyAttempt struct {
	Command  pluginrpc.DshbotSenderPolicyCommand
	Sequence uint64
	Sent     bool
}
type senderPolicyEntry struct {
	state    SenderPolicySnapshot
	attempt  *senderPolicyAttempt
	loading  uint64
	hasDraft bool
}

// Scope-local entries preserve sent requests across hidden/reopened frames. No
// old callback can dispatch, adopt or write status into a replacement account.
type SenderPolicyControls struct {
	mu                   sync.Mutex
	active               RoutineScope
	generation, sequence uint64
	entries              map[RoutineScope]*senderPolicyEntry
}

func (c *SenderPolicyControls) entry(k RoutineScope) *senderPolicyEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*senderPolicyEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &senderPolicyEntry{state: SenderPolicySnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *SenderPolicyControls) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && c.active == k
}
func (c *SenderPolicyControls) Bind(k RoutineScope) {
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
func (c *SenderPolicyControls) Snapshot(k RoutineScope) SenderPolicySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return SenderPolicySnapshot{Scope: k, Generation: c.generation}
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotSenderPolicy(v.Status)
	v.Draft = pluginrpc.CloneDshbotSenderPolicyValues(v.Draft)
	v.Warnings = append([]string(nil), v.Warnings...)
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotSenderPolicyCommand(r.Command)
		v.Review = &r
	}
	return v
}
func (c *SenderPolicyControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *SenderPolicyControls) BeginLoad(k RoutineScope) (SenderPolicyTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return SenderPolicyTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return SenderPolicyTicket{}, false
	}
	c.sequence++
	c.generation++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = ""
	return SenderPolicyTicket{k, c.generation, c.sequence}, true
}
func (c *SenderPolicyControls) LoadCurrent(t SenderPolicyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *SenderPolicyControls) FinishLoad(t SenderPolicyTicket, status pluginrpc.DshbotSenderPolicyStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotSenderPolicyValue(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = storelocale.T(storelocale.KeyStoreDshbotReadUnavailableDraftRetained4)
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotSenderPolicy(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotSenderPolicyApplied(e.attempt.Command, status.DshbotCapabilityStatus) == nil {
			e.state.Candidate = true
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotMatchingProfileFoundInspectAndAdopt4)
		} else {
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownOutcomeRemainsLocked)
		}
		return true
	}
	if !e.hasDraft || !e.state.Dirty || pluginrpc.SameDshbotSenderPolicyValues(e.state.Draft, value) {
		e.state.Draft = pluginrpc.CloneDshbotSenderPolicyValues(value)
	}
	e.hasDraft = true
	e.state.Dirty = !pluginrpc.SameDshbotSenderPolicyValues(e.state.Draft, value)
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadBackReviewRemainingChanges)
	return true
}
func (c *SenderPolicyControls) editable(k RoutineScope) *senderPolicyEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if !e.state.Loaded || !e.state.Writable || e.state.Busy || e.attempt != nil || e.state.Status.WriteError != "" {
		return nil
	}
	return e
}
func (c *SenderPolicyControls) SetDraft(k RoutineScope, v pluginrpc.DshbotSenderPolicyValues) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return
	}
	before, err := pluginrpc.DshbotSenderPolicyValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = pluginrpc.CloneDshbotSenderPolicyValues(v)
	e.state.Dirty = !pluginrpc.SameDshbotSenderPolicyValues(v, before)
	e.hasDraft = true
	c.generation++
}
func (c *SenderPolicyControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	before, err := pluginrpc.DshbotSenderPolicyValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = before
	e.state.Dirty = false
	e.state.Review = nil
	e.state.Error = ""
	c.generation++
}
func (c *SenderPolicyControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Dirty {
		return
	}
	cmd := pluginrpc.DshbotSenderPolicyCommand{Expected: pluginrpc.CloneDshbotSenderPolicy(e.state.Status), Values: e.state.Draft}
	if err := pluginrpc.ValidateDshbotSenderPolicyCommand(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &SenderPolicyConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
}
func (c *SenderPolicyControls) Dismiss(k RoutineScope, r *SenderPolicyConfirmation) {
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
func (c *SenderPolicyControls) Acknowledge(k RoutineScope, r *SenderPolicyConfirmation, ack bool) {
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
func (c *SenderPolicyControls) AcknowledgeExpansion(k RoutineScope, r *SenderPolicyConfirmation, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && r != nil && e.state.Review != nil && r.Sequence == e.state.Review.Sequence {
		e.state.Review.ExpansionAcknowledged = ack
		c.generation++
	}
}
func (c *SenderPolicyControls) BeginSave(k RoutineScope, r *SenderPolicyConfirmation) (SenderPolicyTicket, pluginrpc.DshbotSenderPolicyCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotSenderPolicyCommand
	e := c.editable(k)
	if e == nil || r == nil || e.state.Review == nil || e.state.Review.Sequence != r.Sequence || !e.state.Review.Acknowledged || (pluginrpc.DshbotSenderPolicyExpands(e.state.Review.Command) && !e.state.Review.ExpansionAcknowledged) || !pluginrpc.SameDshbotSenderPolicy(e.state.Status, r.Command.Expected) || !pluginrpc.SameDshbotSenderPolicyValues(e.state.Draft, r.Command.Values) {
		return SenderPolicyTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotSenderPolicyCommand(e.state.Review.Command)
	c.sequence++
	c.generation++
	e.attempt = &senderPolicyAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = ""
	return SenderPolicyTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *SenderPolicyControls) current(t SenderPolicyTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *SenderPolicyControls) Current(t SenderPolicyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *SenderPolicyControls) MarkSent(t SenderPolicyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *SenderPolicyControls) AbortUnsent(t SenderPolicyTicket, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || e.attempt.Sent {
		return
	}
	e.attempt = nil
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotNotSentRefreshAndReview)
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *SenderPolicyControls) FinishSave(t SenderPolicyTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
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
			e.state.Error = err.Error()
		}
		return
	}
	if err == nil {
		err = pluginrpc.DshbotSenderPolicyApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownReadBackOnly)
		return
	}
	e.state.Status.DshbotCapabilityStatus = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = pluginrpc.CloneDshbotSenderPolicyValues(e.attempt.Command.Values)
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotSavedProfileLiveSynchronizationAndE)
	e.attempt = nil
}
func (c *SenderPolicyControls) Adopt(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate || pluginrpc.DshbotSenderPolicyApplied(e.attempt.Command, e.state.Status.DshbotCapabilityStatus) != nil {
		return false
	}
	e.state.Draft = pluginrpc.CloneDshbotSenderPolicyValues(e.attempt.Command.Values)
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Review = nil
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotAdoptedCatalogLiveSynchronizationAnd)
	e.attempt = nil
	c.generation++
	return true
}

package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type TextProfileTicket struct {
	Scope                RoutineScope
	Generation, Sequence uint64
}
type TextProfileConfirmation struct {
	Command      pluginrpc.DshbotTextProfileCommand
	Sequence     uint64
	Acknowledged bool
}
type TextProfileSnapshot struct {
	Scope                                               RoutineScope
	Generation                                          uint64
	Loaded, Busy, Writable, Dirty, Uncertain, Candidate bool
	Draft                                               pluginrpc.DshbotTextProfileValues
	Status                                              pluginrpc.DshbotCapabilityStatus
	Review                                              *TextProfileConfirmation
	Warnings                                            []string
	Error, Notice                                       string
}
type textProfileAttempt struct {
	Command  pluginrpc.DshbotTextProfileCommand
	Sequence uint64
	Sent     bool
}
type textProfileEntry struct {
	state    TextProfileSnapshot
	attempt  *textProfileAttempt
	loading  uint64
	hasDraft bool
}

// Scope-local entries preserve sent requests across hidden/reopened frames. No
// old callback can dispatch, adopt or write status into a replacement account.
type TextProfileControls struct {
	mu                   sync.Mutex
	active               RoutineScope
	generation, sequence uint64
	entries              map[RoutineScope]*textProfileEntry
}

func (c *TextProfileControls) entry(k RoutineScope) *textProfileEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*textProfileEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &textProfileEntry{state: TextProfileSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *TextProfileControls) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && c.active == k
}
func (c *TextProfileControls) Bind(k RoutineScope) {
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
func (c *TextProfileControls) Snapshot(k RoutineScope) TextProfileSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return TextProfileSnapshot{Scope: k, Generation: c.generation}
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotCapabilities(v.Status)
	v.Warnings = append([]string(nil), v.Warnings...)
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotTextProfileCommand(r.Command)
		v.Review = &r
	}
	return v
}
func (c *TextProfileControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *TextProfileControls) BeginLoad(k RoutineScope) (TextProfileTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return TextProfileTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return TextProfileTicket{}, false
	}
	c.sequence++
	c.generation++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = ""
	return TextProfileTicket{k, c.generation, c.sequence}, true
}
func (c *TextProfileControls) LoadCurrent(t TextProfileTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *TextProfileControls) FinishLoad(t TextProfileTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotTextProfileValue(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = storelocale.T(storelocale.KeyStoreDshbotReadUnavailableDraftRetained5)
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotTextProfileApplied(e.attempt.Command, status) == nil {
			e.state.Candidate = true
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotMatchingProfileFoundInspectAndAdopt5)
		} else {
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownOutcomeRemainsLocked)
		}
		return true
	}
	draftValue, draftErr := pluginrpc.ParseDshbotTextProfile(e.state.Draft)
	if !e.hasDraft || !e.state.Dirty || draftErr == nil && draftValue == value {
		e.state.Draft = value
	}
	e.hasDraft = true
	e.state.Dirty = e.state.Draft != value
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadBackReviewRemainingChanges)
	return true
}
func (c *TextProfileControls) editable(k RoutineScope) *textProfileEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if !e.state.Loaded || !e.state.Writable || e.state.Busy || e.attempt != nil || e.state.Status.WriteError != "" {
		return nil
	}
	return e
}
func (c *TextProfileControls) SetDraft(k RoutineScope, v pluginrpc.DshbotTextProfileValues) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return
	}
	before, err := pluginrpc.DshbotTextProfileValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = v
	e.state.Dirty = v != before
	e.hasDraft = true
	c.generation++
}
func (c *TextProfileControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	before, err := pluginrpc.DshbotTextProfileValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = before
	e.state.Dirty = false
	e.state.Review = nil
	e.state.Error = ""
	c.generation++
}
func (c *TextProfileControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Dirty {
		return
	}
	values, err := pluginrpc.ParseDshbotTextProfile(e.state.Draft)
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	cmd := pluginrpc.DshbotTextProfileCommand{Expected: pluginrpc.CloneDshbotCapabilities(e.state.Status), Values: values}
	if err := pluginrpc.ValidateDshbotTextProfileCommand(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &TextProfileConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
}
func (c *TextProfileControls) Dismiss(k RoutineScope, r *TextProfileConfirmation) {
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
func (c *TextProfileControls) Acknowledge(k RoutineScope, r *TextProfileConfirmation, ack bool) {
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
func (c *TextProfileControls) BeginSave(k RoutineScope, r *TextProfileConfirmation) (TextProfileTicket, pluginrpc.DshbotTextProfileCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotTextProfileCommand
	e := c.editable(k)
	if e == nil {
		return TextProfileTicket{}, zero, false
	}
	values, parseErr := pluginrpc.ParseDshbotTextProfile(e.state.Draft)
	if parseErr != nil || r == nil || e.state.Review == nil || e.state.Review.Sequence != r.Sequence || !e.state.Review.Acknowledged || !pluginrpc.SameDshbotCapabilities(e.state.Status, r.Command.Expected) || values != r.Command.Values {
		return TextProfileTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotTextProfileCommand(e.state.Review.Command)
	c.sequence++
	c.generation++
	e.attempt = &textProfileAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = ""
	return TextProfileTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *TextProfileControls) current(t TextProfileTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *TextProfileControls) Current(t TextProfileTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *TextProfileControls) MarkSent(t TextProfileTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *TextProfileControls) AbortUnsent(t TextProfileTicket, err error) {
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
func (c *TextProfileControls) FinishSave(t TextProfileTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
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
		err = pluginrpc.DshbotTextProfileApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownReadBackOnly)
		return
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = e.attempt.Command.Values
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotProfileSavedVerifyPresentationWarnin3)
	e.attempt = nil
}
func (c *TextProfileControls) Adopt(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate || pluginrpc.DshbotTextProfileApplied(e.attempt.Command, e.state.Status) != nil {
		return false
	}
	e.state.Draft = e.attempt.Command.Values
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Review = nil
	e.state.Error = ""
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotAdoptedCatalogPresentationRenameAnd)
	e.attempt = nil
	c.generation++
	return true
}

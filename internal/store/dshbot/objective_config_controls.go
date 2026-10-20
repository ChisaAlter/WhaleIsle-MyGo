package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type ObjectiveConfigTicket struct {
	Scope      RoutineScope
	generation uint64
}
type ObjectiveConfigConfirmation struct {
	Command      pluginrpc.DshbotObjectiveConfigCommand
	Sequence     uint64
	Acknowledged bool
}
type ObjectiveConfigSnapshot struct {
	Scope                         RoutineScope
	Generation                    uint64
	Loaded, Busy, Writable, Dirty bool
	Draft                         pluginrpc.DshbotObjectiveConfigDraft
	Status                        pluginrpc.DshbotCapabilityStatus
	Review                        *ObjectiveConfigConfirmation
	Warnings                      []string
	Error, Notice                 string
}
type ObjectiveConfigControls struct {
	mu                   sync.Mutex
	state                ObjectiveConfigSnapshot
	generation, sequence uint64
	hasDraft             bool
}

func (c *ObjectiveConfigControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		c.generation++
		c.sequence++
		c.state = ObjectiveConfigSnapshot{Scope: k}
		c.hasDraft = false
	}
}
func (c *ObjectiveConfigControls) valid(k RoutineScope) bool {
	return c.state.Scope == k && k.HostKey != "" && k.BotID != ""
}
func (c *ObjectiveConfigControls) Snapshot(k RoutineScope) ObjectiveConfigSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		return ObjectiveConfigSnapshot{Scope: k}
	}
	out := c.state
	out.Generation = c.generation
	out.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	out.Warnings = append([]string(nil), out.Warnings...)
	if out.Review != nil {
		r := *out.Review
		r.Command.Expected = pluginrpc.CloneDshbotCapabilities(r.Command.Expected)
		out.Review = &r
	}
	return out
}
func (c *ObjectiveConfigControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *ObjectiveConfigControls) current(t ObjectiveConfigTicket) bool {
	return c.valid(t.Scope) && t.generation == c.generation && c.state.Busy
}
func (c *ObjectiveConfigControls) Current(t ObjectiveConfigTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *ObjectiveConfigControls) BeginLoad(k RoutineScope) (ObjectiveConfigTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.state.Busy {
		return ObjectiveConfigTicket{}, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	c.state.Review = nil
	c.state.Error = ""
	return ObjectiveConfigTicket{k, c.generation}, true
}
func (c *ObjectiveConfigControls) FinishLoad(t ObjectiveConfigTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.generation++
	c.state.Busy = false
	value, valueErr := pluginrpc.DshbotObjectiveConfigValue(status)
	if err != nil || valueErr != nil || status.Bot.ID != t.Scope.BotID || status.Bot.SessionID == "" {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotReadUnavailableDraftRetainedAndLock2)
		return true
	}
	draftValue, draftErr := pluginrpc.ParseDshbotObjectiveConfigDraft(c.state.Draft)
	if !c.hasDraft || !c.state.Dirty || draftErr == nil && draftValue == value {
		c.state.Draft = value.Draft()
	}
	c.hasDraft = true
	c.state.Dirty = c.state.Draft != value.Draft()
	c.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	c.state.Writable = writable
	c.state.Loaded = true
	c.state.Error = ""
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadBackReviewAnyRemainingDraft)
	return true
}
func (c *ObjectiveConfigControls) editable(k RoutineScope) bool {
	return c.valid(k) && c.state.Loaded && c.state.Writable && !c.state.Busy && c.state.Status.WriteError == ""
}
func (c *ObjectiveConfigControls) SetDraft(k RoutineScope, draft pluginrpc.DshbotObjectiveConfigDraft) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || c.state.Review != nil {
		return
	}
	current, err := pluginrpc.DshbotObjectiveConfigValue(c.state.Status)
	if err != nil {
		return
	}
	c.state.Draft = draft
	c.state.Dirty = draft != current.Draft()
	c.generation++
}
func (c *ObjectiveConfigControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.state.Loaded || c.state.Busy {
		return
	}
	value, err := pluginrpc.DshbotObjectiveConfigValue(c.state.Status)
	if err != nil {
		return
	}
	c.state.Draft = value.Draft()
	c.state.Dirty = false
	c.state.Review = nil
	c.generation++
}
func (c *ObjectiveConfigControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || !c.state.Dirty {
		return
	}
	desired, err := pluginrpc.ParseDshbotObjectiveConfigDraft(c.state.Draft)
	if err != nil {
		c.state.Error = err.Error()
		return
	}
	cmd := pluginrpc.DshbotObjectiveConfigCommand{Expected: pluginrpc.CloneDshbotCapabilities(c.state.Status), Desired: desired}
	if err := pluginrpc.ValidateDshbotObjectiveConfigCommand(cmd); err != nil {
		c.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	c.state.Review = &ObjectiveConfigConfirmation{Command: cmd, Sequence: c.sequence}
	c.state.Error = ""
}
func (c *ObjectiveConfigControls) Dismiss(k RoutineScope, r *ObjectiveConfigConfirmation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review = nil
		c.generation++
	}
}
func (c *ObjectiveConfigControls) Acknowledge(k RoutineScope, r *ObjectiveConfigConfirmation, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review.Acknowledged = ack
		c.generation++
	}
}
func (c *ObjectiveConfigControls) BeginSave(k RoutineScope, r *ObjectiveConfigConfirmation) (ObjectiveConfigTicket, pluginrpc.DshbotObjectiveConfigCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotObjectiveConfigCommand
	actual := c.state.Review
	if !c.editable(k) || r == nil || actual == nil || actual.Sequence != r.Sequence || !actual.Acknowledged || !pluginrpc.SameDshbotCapabilities(actual.Command.Expected, c.state.Status) {
		return ObjectiveConfigTicket{}, zero, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	cmd := actual.Command
	cmd.Expected = pluginrpc.CloneDshbotCapabilities(cmd.Expected)
	return ObjectiveConfigTicket{k, c.generation}, cmd, true
}
func (c *ObjectiveConfigControls) FinishSave(t ObjectiveConfigTicket, out pluginrpc.DshbotCapabilityReceipt, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.state.Review == nil {
		return false
	}
	c.generation++
	c.state.Busy = false
	c.state.Review = nil
	c.state.Warnings = append([]string(nil), out.Warnings...)
	if err != nil || out.Status.Bot.ID != t.Scope.BotID || out.Status.Bot.SessionID != c.state.Status.Bot.SessionID {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotOutcomeUncertainReadBackBeforeSavin2)
		if !out.Attempted {
			c.state.Error = storelocale.T(storelocale.KeyStoreDshbotPreflightChangedNoSaveSent2)
		}
		return true
	}
	c.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	c.state.Error = ""
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReceiptReceivedReadBackToVerify)
	return true
}

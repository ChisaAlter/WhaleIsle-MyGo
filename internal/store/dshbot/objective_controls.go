package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type ObjectiveTicket struct {
	Scope      RoutineScope
	generation uint64
}
type ObjectiveReview struct {
	Command  pluginrpc.DshbotObjectiveCommand
	Sequence uint64
}
type ObjectiveSnapshot struct {
	Scope         RoutineScope
	Generation    uint64
	Loaded, Busy  bool
	Status        pluginrpc.DshbotObjectiveStatus
	Review        *ObjectiveReview
	Warnings      []string
	Error, Notice string
}
type ObjectiveControls struct {
	mu                   sync.Mutex
	state                ObjectiveSnapshot
	generation, sequence uint64
}

func (c *ObjectiveControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		c.generation++
		c.sequence++
		c.state = ObjectiveSnapshot{Scope: k}
	}
}
func (c *ObjectiveControls) valid(k RoutineScope) bool {
	return k == c.state.Scope && k.HostKey != "" && k.BotID != ""
}
func (c *ObjectiveControls) Snapshot(k RoutineScope) ObjectiveSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		return ObjectiveSnapshot{Scope: k}
	}
	out := c.state
	out.Generation = c.generation
	out.Status = pluginrpc.CloneDshbotObjective(out.Status)
	out.Warnings = append([]string(nil), out.Warnings...)
	if out.Review != nil {
		r := *out.Review
		r.Command.Expected = pluginrpc.CloneDshbotObjective(r.Command.Expected)
		out.Review = &r
	}
	return out
}
func (c *ObjectiveControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *ObjectiveControls) current(t ObjectiveTicket) bool {
	return c.valid(t.Scope) && t.generation == c.generation && c.state.Busy
}
func (c *ObjectiveControls) Current(t ObjectiveTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *ObjectiveControls) BeginLoad(k RoutineScope) (ObjectiveTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.state.Busy {
		return ObjectiveTicket{}, false
	}
	c.generation++
	c.state.Loaded = false
	c.state.Busy = true
	c.state.Review = nil
	c.state.Error = ""
	return ObjectiveTicket{k, c.generation}, true
}
func (c *ObjectiveControls) FinishLoad(t ObjectiveTicket, value pluginrpc.DshbotObjectiveStatus, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.generation++
	c.state.Busy = false
	if err != nil || value.Bot.ID != t.Scope.BotID || value.Bot.SessionID == "" || !value.GoalKnown {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotObjectiveStateUnavailableRefreshBefo)
		return true
	}
	c.state.Status = pluginrpc.CloneDshbotObjective(value)
	c.state.Loaded = true
	c.state.Error = ""
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRefreshedWithoutStartingABot)
	return true
}
func (c *ObjectiveControls) Prepare(k RoutineScope, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.state.Loaded || c.state.Busy {
		return
	}
	cmd := pluginrpc.DshbotObjectiveCommand{Action: action, Expected: pluginrpc.CloneDshbotObjective(c.state.Status)}
	if err := pluginrpc.ValidateDshbotObjectiveCommand(cmd); err != nil {
		c.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	c.state.Review = &ObjectiveReview{Command: cmd, Sequence: c.sequence}
	c.state.Error = ""
}
func (c *ObjectiveControls) Dismiss(k RoutineScope, r *ObjectiveReview) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.generation++
		c.state.Review = nil
	}
}
func (c *ObjectiveControls) BeginAction(k RoutineScope, r *ObjectiveReview) (ObjectiveTicket, pluginrpc.DshbotObjectiveCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotObjectiveCommand
	actual := c.state.Review
	if !c.valid(k) || !c.state.Loaded || c.state.Busy || r == nil || actual == nil || r.Sequence != actual.Sequence || !pluginrpc.SameDshbotObjective(actual.Command.Expected, c.state.Status) {
		return ObjectiveTicket{}, zero, false
	}
	c.generation++
	c.state.Loaded = false
	c.state.Busy = true
	cmd := actual.Command
	cmd.Expected = pluginrpc.CloneDshbotObjective(cmd.Expected)
	return ObjectiveTicket{k, c.generation}, cmd, true
}
func (c *ObjectiveControls) FinishAction(t ObjectiveTicket, result pluginrpc.DshbotObjectiveResult, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.state.Review == nil {
		return false
	}
	c.generation++
	c.state.Busy = false
	c.state.Review = nil
	c.state.Warnings = append([]string(nil), result.Warnings...)
	// Every action needs an independent read before another action. A receipt is
	// not proof of work completion, and clear omits the live goal altogether.
	if err != nil || result.Status.Bot.ID != t.Scope.BotID || result.Status.Bot.SessionID != c.state.Status.Bot.SessionID {
		c.state.Error = storelocale.T(storelocale.KeyStoreDshbotOutcomeUncertainRefreshBeforeAnother)
		if !result.Attempted {
			c.state.Error = storelocale.T(storelocale.KeyStoreDshbotPreflightChangedOrUnavailableNoActi)
		}
		return true
	}
	c.state.Status = pluginrpc.CloneDshbotObjective(result.Status)
	c.state.Error = ""
	c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotHostReceiptReceivedRefreshToVerify)
	return true
}

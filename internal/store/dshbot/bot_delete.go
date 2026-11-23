package dshbot

import (
	"fmt"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type BotDeleteReview struct {
	Plan               pluginrpc.DshbotBotDeletePlan
	Sequence           uint64
	TypedID, TypedName string
	Acknowledged       bool
}
type BotDeleteSnapshot struct {
	Scope                                        string
	Generation                                   uint64
	Loaded, Writable, Busy, Uncertain, Completed bool
	Plan                                         pluginrpc.DshbotBotDeletePlan
	Review                                       *BotDeleteReview
	Pending                                      *pluginrpc.DshbotBotDeleteCommand
	Observation                                  string
	Warnings                                     []string
	Error                                        *state.Fault
	Notice                                       string
}
type BotDeleteTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type botDeleteAttempt struct {
	Command  pluginrpc.DshbotBotDeleteCommand
	Sequence uint64
	Sent     bool
}
type botDeleteEntry struct {
	state   BotDeleteSnapshot
	attempt *botDeleteAttempt
	loading uint64
	pub     state.Published[BotDeleteSnapshot]
	dirty   bool
}

// Unknown attempts stay locked for the life of this process. No catalog read,
// cancellation, navigation or client replacement can turn absence into a receipt.
type BotDeleter struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*botDeleteEntry
}

func (c *BotDeleter) entry(k string) *botDeleteEntry {
	if c.entries == nil {
		c.entries = map[string]*botDeleteEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &botDeleteEntry{state: BotDeleteSnapshot{Scope: k}}
	}
	c.entries[k].dirty = true
	return c.entries[k]
}
func (c *BotDeleter) valid(k string) bool { return k != "" && c.active == k && c.binding != "" }
func (c *BotDeleter) Bind(k, binding string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == k && c.binding == binding {
		return
	}
	c.generation++
	c.active = k
	c.binding = binding
	if k != "" {
		e := c.entry(k)
		e.state.Loaded = false
		e.state.Review = nil
		e.loading = 0
		if e.attempt == nil || e.state.Uncertain {
			e.state.Busy = false
		}
	}
}
func (c *BotDeleter) Suspend() { c.Bind("", "") }
func (c *BotDeleter) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}

// Snapshot returns the published view for scope k; unchanged reads share
// the version and allocate nothing.
func (c *BotDeleter) Snapshot(k string) *state.Version[BotDeleteSnapshot] {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		e = c.entry(k)
	}
	if v := e.pub.Load(); v != nil && !e.dirty {
		return v
	}
	out := e.state
	out.Generation = c.generation
	out.Plan = pluginrpc.CloneDshbotBotDeletePlan(out.Plan)
	out.Warnings = append([]string(nil), out.Warnings...)
	if out.Review != nil {
		r := *out.Review
		r.Plan = pluginrpc.CloneDshbotBotDeletePlan(r.Plan)
		out.Review = &r
	}
	if e.attempt != nil {
		p := pluginrpc.CloneDshbotBotDeleteCommand(e.attempt.Command)
		out.Pending = &p
	}
	e.pub.Publish(out)
	e.dirty = false
	return e.pub.Load()
}
func (c *BotDeleter) BeginLoad(k string) (BotDeleteTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return BotDeleteTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy || e.state.Completed {
		return BotDeleteTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = nil
	return BotDeleteTicket{k, c.generation, c.sequence}, true
}
func (c *BotDeleter) FinishLoad(t BotDeleteTicket, p pluginrpc.DshbotBotDeletePlan, v pluginrpc.DshbotDeleteCatalog, writable bool, err error) bool {
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
	e.state.Writable = writable
	if err != nil {
		e.state.Error = state.FaultOf(err)
		return true
	}
	e.state.Loaded = true
	e.state.Error = nil
	if e.attempt != nil && e.state.Uncertain {
		e.state.Observation = pluginrpc.DshbotBotDeleteObservation(e.attempt.Command.Plan, v)
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotReadbackDoesNotSettleTheUnknownAtt)
		return true
	}
	e.state.Plan = pluginrpc.CloneDshbotBotDeletePlan(p)
	e.state.Warnings = append([]string(nil), p.Preview.Warnings...)
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotPreviewReadDeletionNotAuthorized)
	return true
}
func (c *BotDeleter) Prepare(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.attempt != nil || e.state.Completed {
		return
	}
	p, err := pluginrpc.BuildDshbotBotDeletePlan(e.state.Plan.Catalog, e.state.Plan.Preview, e.state.Plan.Target.ID)
	if err != nil || !pluginrpc.SameDshbotBotDeletePlan(p, e.state.Plan) {
		e.state.Error = state.FaultText(storelocale.T(storelocale.KeyStoreDshbotDeletionPreviewExpired))
		return
	}
	c.sequence++
	e.state.Review = &BotDeleteReview{Plan: p, Sequence: c.sequence}
	e.state.Error = nil
}
func sameBotDeleteReview(a, b *BotDeleteReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && pluginrpc.SameDshbotBotDeletePlan(a.Plan, b.Plan)
}
func (c *BotDeleter) ConfirmIdentity(k string, r *BotDeleteReview, id, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && sameBotDeleteReview(e.state.Review, r) {
		e.state.Review.TypedID = id
		e.state.Review.TypedName = name
	}
}
func (c *BotDeleter) Acknowledge(k string, r *BotDeleteReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && sameBotDeleteReview(e.state.Review, r) {
		e.state.Review.Acknowledged = ack
	}
}
func (c *BotDeleter) Cancel(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.attempt == nil && !e.state.Busy {
		e.state.Review = nil
		e.state.Loaded = false
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotCancelledBeforeDispatch)
	}
}
func (c *BotDeleter) Begin(k string, r *BotDeleteReview) (BotDeleteTicket, pluginrpc.DshbotBotDeleteCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotBotDeleteCommand
	if !c.valid(k) {
		return BotDeleteTicket{}, empty, false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Uncertain || e.state.Completed || !e.state.Loaded || !e.state.Writable || !sameBotDeleteReview(e.state.Review, r) {
		return BotDeleteTicket{}, empty, false
	}
	live := e.state.Review
	cmd := pluginrpc.DshbotBotDeleteCommand{Plan: pluginrpc.CloneDshbotBotDeletePlan(live.Plan), ConfirmedID: live.TypedID, ConfirmedName: live.TypedName, Acknowledged: live.Acknowledged}
	if err := pluginrpc.ValidateDshbotBotDelete(cmd); err != nil {
		e.state.Error = state.FaultOf(err)
		return BotDeleteTicket{}, empty, false
	}
	c.sequence++
	e.attempt = &botDeleteAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = nil
	return BotDeleteTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *BotDeleter) current(t BotDeleteTicket) bool {
	e := c.entries[t.Scope]
	if e != nil {
		e.dirty = true
	}
	return c.valid(t.Scope) && t.Generation == c.generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *BotDeleter) Current(t BotDeleteTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *BotDeleter) MarkSent(t BotDeleteTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	a := c.entries[t.Scope].attempt
	if a.Sent {
		return false
	}
	a.Sent = true
	return true
}
func (c *BotDeleter) AbortUnsent(t BotDeleteTicket, err error) {
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
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotDeletionNotSent)
	if err != nil {
		e.state.Error = state.FaultOf(err)
	}
}
func (c *BotDeleter) Finish(t BotDeleteTicket, result pluginrpc.DshbotBotDeleteResult, err error) {
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
	e.state.Warnings = append([]string(nil), result.Warnings...)
	if err != nil && result.Rejected {
		e.attempt = nil
		e.state.Error = state.FaultOf(err)
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRejectedBeforeEffects)
		return
	}
	if err == nil {
		err = pluginrpc.ValidateDshbotBotDeleteReceipt(e.attempt.Command, result)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = state.FaultOf(err)
		e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotUnknownSideEffectsMayAlreadyHaveOc)
		return
	}
	e.state.Plan = pluginrpc.CloneDshbotBotDeletePlan(e.attempt.Command.Plan)
	e.state.Completed = true
	e.state.Uncertain = false
	e.state.Error = nil
	e.state.Notice = fmt.Sprintf(storelocale.T(storelocale.KeyStoreDshbotDeletionConfirmed), result.Removed)
	e.attempt = nil
}

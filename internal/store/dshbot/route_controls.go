package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type RouteTicket struct {
	Scope                RoutineScope
	Generation, Sequence uint64
}
type RouteConfirmation struct {
	Command      pluginrpc.DshbotRouteCommand
	Sequence     uint64
	Acknowledged bool
}
type RouteSnapshot struct {
	Scope                                               RoutineScope
	Generation                                          uint64
	Loaded, Busy, Writable, Dirty, Uncertain, Candidate bool
	Draft                                               pluginrpc.DshbotRouteValues
	Status                                              pluginrpc.DshbotRouteStatus
	Review                                              *RouteConfirmation
	Warnings                                            []string
	Error, Notice                                       string
}
type routeAttempt struct {
	Command  pluginrpc.DshbotRouteCommand
	Sequence uint64
	Sent     bool
}
type routeEntry struct {
	state    RouteSnapshot
	attempt  *routeAttempt
	loading  uint64
	hasDraft bool
}

// Scope-local entries preserve sent requests across hidden/reopened frames. No
// old callback can dispatch, adopt or write status into a replacement account.
type RouteControls struct {
	mu                   sync.Mutex
	active               RoutineScope
	generation, sequence uint64
	entries              map[RoutineScope]*routeEntry
}

func (c *RouteControls) entry(k RoutineScope) *routeEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*routeEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &routeEntry{state: RouteSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *RouteControls) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && c.active == k
}
func (c *RouteControls) Bind(k RoutineScope) {
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
func (c *RouteControls) Snapshot(k RoutineScope) RouteSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return RouteSnapshot{Scope: k, Generation: c.generation}
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotRoute(v.Status)
	v.Warnings = append([]string(nil), v.Warnings...)
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotRouteCommand(r.Command)
		v.Review = &r
	}
	return v
}
func (c *RouteControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *RouteControls) BeginLoad(k RoutineScope) (RouteTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return RouteTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return RouteTicket{}, false
	}
	c.sequence++
	c.generation++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = ""
	return RouteTicket{k, c.generation, c.sequence}, true
}
func (c *RouteControls) LoadCurrent(t RouteTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *RouteControls) FinishLoad(t RouteTicket, status pluginrpc.DshbotRouteStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotRouteValue(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = "模型/工作区未核实；保留草稿，请刷新 / Read unavailable; draft retained"
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotRoute(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotRouteApplied(e.attempt.Command, status.DshbotCapabilityStatus) == nil {
			e.state.Candidate = true
			e.state.Notice = "完整资料已匹配审阅值，核对新会话后可采用读回 / Matching profile found; inspect and adopt"
		} else {
			e.state.Notice = "结果仍未知；仅保留读回，不允许重发。请在 Bot Web 核对 / Unknown outcome remains locked"
		}
		return true
	}
	if !e.hasDraft || !e.state.Dirty || e.state.Draft == value {
		e.state.Draft = value
	}
	e.hasDraft = true
	e.state.Dirty = e.state.Draft != value
	e.state.Notice = "已读回；剩余草稿需要重新审阅 / Read back; review remaining changes"
	return true
}
func (c *RouteControls) editable(k RoutineScope) *routeEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if !e.state.Loaded || !e.state.Writable || e.state.Busy || e.attempt != nil || e.state.Status.WriteError != "" {
		return nil
	}
	return e
}
func (c *RouteControls) SetDraft(k RoutineScope, v pluginrpc.DshbotRouteValues) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return
	}
	before, err := pluginrpc.DshbotRouteValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = v
	e.state.Dirty = v != before
	e.hasDraft = true
	c.generation++
}
func (c *RouteControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	before, err := pluginrpc.DshbotRouteValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = before
	e.state.Dirty = false
	e.state.Review = nil
	e.state.Error = ""
	c.generation++
}
func (c *RouteControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Dirty {
		return
	}
	cmd := pluginrpc.DshbotRouteCommand{Expected: pluginrpc.CloneDshbotRoute(e.state.Status), Values: e.state.Draft}
	if err := pluginrpc.ValidateDshbotRouteCommand(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &RouteConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
}
func (c *RouteControls) Dismiss(k RoutineScope, r *RouteConfirmation) {
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
func (c *RouteControls) Acknowledge(k RoutineScope, r *RouteConfirmation, ack bool) {
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
func (c *RouteControls) BeginSave(k RoutineScope, r *RouteConfirmation) (RouteTicket, pluginrpc.DshbotRouteCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotRouteCommand
	e := c.editable(k)
	if e == nil || r == nil || e.state.Review == nil || e.state.Review.Sequence != r.Sequence || !e.state.Review.Acknowledged || !pluginrpc.SameDshbotRoute(e.state.Status, r.Command.Expected) || e.state.Draft != r.Command.Values {
		return RouteTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotRouteCommand(e.state.Review.Command)
	c.sequence++
	c.generation++
	e.attempt = &routeAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = ""
	return RouteTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *RouteControls) current(t RouteTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *RouteControls) Current(t RouteTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *RouteControls) MarkSent(t RouteTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *RouteControls) AbortUnsent(t RouteTicket, err error) {
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
	e.state.Notice = "未发送；草稿保留，刷新重新审阅 / Not sent; refresh and review"
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *RouteControls) FinishSave(t RouteTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
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
		e.state.Notice = "未发送；刷新重新审阅 / Not sent; refresh"
		if err != nil {
			e.state.Error = err.Error()
		}
		return
	}
	if err == nil {
		err = pluginrpc.DshbotRouteApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "结果未知；保留并锁定草稿，只读回，不自动重试 / Unknown; read back only"
		return
	}
	e.state.Status.DshbotCapabilityStatus = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = e.attempt.Command.Values
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = ""
	e.state.Notice = "Host 已返回完整保存回执；尚未打开新会话，刷新后可从现有入口打开 / Saved; use existing entry to open after refresh"
	e.attempt = nil
}
func (c *RouteControls) Adopt(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate || pluginrpc.DshbotRouteApplied(e.attempt.Command, e.state.Status.DshbotCapabilityStatus) != nil {
		return false
	}
	e.state.Draft = e.attempt.Command.Values
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Review = nil
	e.state.Error = ""
	e.state.Notice = "已采用完整匹配资料，没有重发；会话创建及目标同步需另行核实 / Adopted catalog; session and live synchronization remain unverified"
	e.attempt = nil
	c.generation++
	return true
}

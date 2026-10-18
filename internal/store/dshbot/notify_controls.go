package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
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
	Error, Notice                                       string
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
func (c *NotifyControls) Snapshot(k RoutineScope) NotifySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return NotifySnapshot{Scope: k, Generation: c.generation}
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
	return v
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
	e.state.Error = ""
	return NotifyTicket{k, c.generation, c.sequence}, true
}
func (c *NotifyControls) LoadCurrent(t NotifyTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *NotifyControls) FinishLoad(t NotifyTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotNotifyEnabled(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = "通知配置未核实；保留草稿，请刷新 / Read unavailable; draft retained"
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotNotifyApplied(e.attempt.Command, status) == nil {
			e.state.Candidate = true
			e.state.Notice = "完整资料已匹配审阅值，核对通知开关及Host/Bot身份后可采用读回 / Matching profile found; inspect and adopt"
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
	e.state.Error = ""
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
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &NotifyConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
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
	e.state.Error = ""
	return NotifyTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *NotifyControls) current(t NotifyTicket) bool {
	e := c.entries[t.Scope]
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
func (c *NotifyControls) FinishSave(t NotifyTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
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
		err = pluginrpc.DshbotNotifyApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "结果未知；保留并锁定草稿，只读回，不自动重试 / Unknown; read back only"
		return
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = e.attempt.Command.Enabled
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = ""
	e.state.Notice = "Host 已返回完整通知配置回执；请刷新核对展示、警告与目标状态 / Profile saved; verify presentation, warnings and goal state"
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
	e.state.Error = ""
	e.state.Notice = "已采用完整匹配资料，没有重发；会话展示/重命名及目标同步需另行核实 / Adopted catalog; presentation, rename and goal sync remain unverified"
	e.attempt = nil
	c.generation++
	return true
}

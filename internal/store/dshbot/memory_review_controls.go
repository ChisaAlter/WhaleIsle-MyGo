package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type MemoryReviewTicket struct {
	Scope      RoutineScope
	generation uint64
}
type MemoryReviewConfirmation struct {
	Command      pluginrpc.DshbotMemoryReviewCommand
	Sequence     uint64
	Acknowledged bool
}
type MemoryReviewSnapshot struct {
	Scope                                RoutineScope
	Generation                           uint64
	Loaded, Busy, Writable, Dirty, Draft bool
	Status                               pluginrpc.DshbotCapabilityStatus
	Review                               *MemoryReviewConfirmation
	Warnings                             []string
	Error, Notice                        string
}
type MemoryReviewControls struct {
	mu                   sync.Mutex
	state                MemoryReviewSnapshot
	generation, sequence uint64
	hasDraft             bool
}

func (c *MemoryReviewControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		c.generation++
		c.sequence++
		c.state = MemoryReviewSnapshot{Scope: k}
		c.hasDraft = false
	}
}
func (c *MemoryReviewControls) valid(k RoutineScope) bool {
	return c.state.Scope == k && k.HostKey != "" && k.BotID != ""
}
func (c *MemoryReviewControls) Snapshot(k RoutineScope) MemoryReviewSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		return MemoryReviewSnapshot{Scope: k}
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
func (c *MemoryReviewControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *MemoryReviewControls) current(t MemoryReviewTicket) bool {
	return c.valid(t.Scope) && t.generation == c.generation && c.state.Busy
}
func (c *MemoryReviewControls) Current(t MemoryReviewTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *MemoryReviewControls) BeginLoad(k RoutineScope) (MemoryReviewTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.state.Busy {
		return MemoryReviewTicket{}, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	c.state.Review = nil
	c.state.Error = ""
	return MemoryReviewTicket{k, c.generation}, true
}
func (c *MemoryReviewControls) FinishLoad(t MemoryReviewTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.generation++
	c.state.Busy = false
	enabled, flagErr := pluginrpc.DshbotMemoryReviewEnabled(status)
	if err != nil || flagErr != nil || status.Bot.ID != t.Scope.BotID || status.Bot.SessionID == "" {
		c.state.Error = "自动记忆配置未能核实；保留并锁定草稿，请刷新 / Read unavailable; draft retained and locked"
		return true
	}
	if !c.hasDraft || !c.state.Dirty {
		c.state.Draft = enabled
	}
	c.hasDraft = true
	c.state.Dirty = c.state.Draft != enabled
	c.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	c.state.Writable = writable
	c.state.Loaded = true
	c.state.Error = ""
	c.state.Notice = "已读回配置；不同于 Host 的草稿须重新审阅 / Read back; review any remaining draft"
	return true
}
func (c *MemoryReviewControls) editable(k RoutineScope) bool {
	return c.valid(k) && c.state.Loaded && c.state.Writable && !c.state.Busy && c.state.Status.WriteError == ""
}
func (c *MemoryReviewControls) SetEnabled(k RoutineScope, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || c.state.Review != nil {
		return
	}
	current, err := pluginrpc.DshbotMemoryReviewEnabled(c.state.Status)
	if err != nil {
		return
	}
	c.state.Draft = enabled
	c.state.Dirty = enabled != current
	c.generation++
}
func (c *MemoryReviewControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.state.Loaded || c.state.Busy {
		return
	}
	enabled, err := pluginrpc.DshbotMemoryReviewEnabled(c.state.Status)
	if err != nil {
		return
	}
	c.state.Draft = enabled
	c.state.Dirty = false
	c.state.Review = nil
	c.generation++
}
func (c *MemoryReviewControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || !c.state.Dirty {
		return
	}
	cmd := pluginrpc.DshbotMemoryReviewCommand{Expected: pluginrpc.CloneDshbotCapabilities(c.state.Status), Enabled: c.state.Draft}
	if err := pluginrpc.ValidateDshbotMemoryReviewCommand(cmd); err != nil {
		c.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	c.state.Review = &MemoryReviewConfirmation{Command: cmd, Sequence: c.sequence}
	c.state.Error = ""
}
func (c *MemoryReviewControls) Dismiss(k RoutineScope, r *MemoryReviewConfirmation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review = nil
		c.generation++
	}
}
func (c *MemoryReviewControls) Acknowledge(k RoutineScope, r *MemoryReviewConfirmation, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review.Acknowledged = ack
		c.generation++
	}
}
func (c *MemoryReviewControls) BeginSave(k RoutineScope, r *MemoryReviewConfirmation) (MemoryReviewTicket, pluginrpc.DshbotMemoryReviewCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotMemoryReviewCommand
	actual := c.state.Review
	if !c.editable(k) || r == nil || actual == nil || actual.Sequence != r.Sequence || !actual.Acknowledged || !pluginrpc.SameDshbotCapabilities(actual.Command.Expected, c.state.Status) {
		return MemoryReviewTicket{}, zero, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	cmd := actual.Command
	cmd.Expected = pluginrpc.CloneDshbotCapabilities(cmd.Expected)
	return MemoryReviewTicket{k, c.generation}, cmd, true
}
func (c *MemoryReviewControls) FinishSave(t MemoryReviewTicket, out pluginrpc.DshbotCapabilityReceipt, err error) bool {
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
		c.state.Error = "保存结果未知；保留并锁定草稿，先读回，不自动重试 / Outcome uncertain; read back before saving"
		if !out.Attempted {
			c.state.Error = "发送前资料或权限变化，未保存；刷新重新审阅 / Preflight changed; no save sent"
		}
		return true
	}
	c.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	c.state.Error = ""
	c.state.Notice = "Host 返回保存回执；刷新核实配置与警告 / Receipt received; read back to verify"
	return true
}

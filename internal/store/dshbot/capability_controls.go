package dshbot

import (
	"reflect"
	"slices"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type CapabilityTicket struct {
	Scope      RoutineScope
	generation uint64
}
type CapabilityReview struct {
	Command      pluginrpc.DshbotCapabilityCommand
	Sequence     uint64
	Acknowledged bool
}
type CapabilitySnapshot struct {
	Scope                         RoutineScope
	Generation                    uint64
	Loaded, Busy, Writable, Dirty bool
	Status                        pluginrpc.DshbotCapabilityStatus
	Draft                         pluginrpc.DshbotCapabilityPolicies
	Review                        *CapabilityReview
	Warnings                      []string
	Error, Notice                 string
}
type CapabilityControls struct {
	mu                   sync.Mutex
	state                CapabilitySnapshot
	generation, sequence uint64
}

func (c *CapabilityControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		c.generation++
		c.sequence++
		c.state = CapabilitySnapshot{Scope: k}
	}
}
func (c *CapabilityControls) valid(k RoutineScope) bool {
	return c.state.Scope == k && k.HostKey != "" && k.BotID != ""
}
func (c *CapabilityControls) Snapshot(k RoutineScope) CapabilitySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != k {
		return CapabilitySnapshot{Scope: k}
	}
	out := c.state
	out.Generation = c.generation
	out.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	out.Draft = pluginrpc.CloneDshbotCapabilityPolicies(out.Draft)
	out.Warnings = append([]string(nil), out.Warnings...)
	if out.Review != nil {
		r := *out.Review
		r.Command.Expected = pluginrpc.CloneDshbotCapabilities(r.Command.Expected)
		r.Command.Desired = pluginrpc.CloneDshbotCapabilityPolicies(r.Command.Desired)
		out.Review = &r
	}
	return out
}
func (c *CapabilityControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *CapabilityControls) current(t CapabilityTicket) bool {
	return c.valid(t.Scope) && c.generation == t.generation && c.state.Busy
}
func (c *CapabilityControls) Current(t CapabilityTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *CapabilityControls) BeginLoad(k RoutineScope) (CapabilityTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || c.state.Busy {
		return CapabilityTicket{}, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	c.state.Review = nil
	c.state.Error = ""
	return CapabilityTicket{k, c.generation}, true
}
func (c *CapabilityControls) FinishLoad(t CapabilityTicket, value pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.generation++
	c.state.Busy = false
	if err != nil || value.Bot.ID != t.Scope.BotID || value.Bot.SessionID == "" {
		c.state.Error = "未能核实当前能力资料；草稿保留且禁写，请刷新 / Read failed; draft retained and locked"
		return true
	}
	if c.state.Draft == nil || !c.state.Dirty {
		c.state.Draft = pluginrpc.CloneDshbotCapabilityPolicies(value.Policies)
	}
	c.state.Status = pluginrpc.CloneDshbotCapabilities(value)
	c.state.Writable = writable
	c.state.Loaded = true
	c.state.Dirty = !reflect.DeepEqual(c.state.Draft, value.Policies)
	c.state.Error = ""
	c.state.Notice = "已读回当前 Host 资料；保留的草稿需按当前版本重新审阅 / Read back; review preserved draft against current state"
	return true
}
func (c *CapabilityControls) editable(k RoutineScope) bool {
	return c.valid(k) && c.state.Loaded && c.state.Writable && !c.state.Busy && c.state.Status.WriteError == "" && c.state.Status.Catalog.Active
}
func (c *CapabilityControls) Change(k RoutineScope, group, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || c.state.Review != nil || !slices.Contains(pluginrpc.DshbotCapabilityGroups, group) {
		return
	}
	p := c.state.Draft[group]
	switch field {
	case "mode":
		if value != "all" && value != "selected" {
			return
		}
		p.Mode = value
	case "toggle":
		if p.Mode != "selected" {
			return
		}
		if i := slices.Index(p.Names, value); i >= 0 {
			p.Names = append(p.Names[:i], p.Names[i+1:]...)
		} else {
			found := false
			for _, e := range c.state.Status.Catalog.Entries(group) {
				found = found || e.Name == value
			}
			if !found {
				return
			}
			p.Names = append(p.Names, value)
		}
	default:
		return
	}
	c.state.Draft[group] = p
	c.state.Dirty = !reflect.DeepEqual(c.state.Draft, c.state.Status.Policies)
	c.state.Error = ""
	c.generation++
}
func (c *CapabilityControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.state.Loaded || c.state.Busy {
		return
	}
	c.state.Draft = pluginrpc.CloneDshbotCapabilityPolicies(c.state.Status.Policies)
	c.state.Dirty = false
	c.state.Review = nil
	c.generation++
}
func (c *CapabilityControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.editable(k) || !c.state.Dirty {
		return
	}
	cmd := pluginrpc.DshbotCapabilityCommand{Expected: pluginrpc.CloneDshbotCapabilities(c.state.Status), Desired: pluginrpc.CloneDshbotCapabilityPolicies(c.state.Draft)}
	if err := pluginrpc.ValidateDshbotCapabilityCommand(cmd); err != nil {
		c.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	c.state.Review = &CapabilityReview{Command: cmd, Sequence: c.sequence}
	c.state.Error = ""
}
func (c *CapabilityControls) Dismiss(k RoutineScope, r *CapabilityReview) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review = nil
		c.generation++
	}
}
func (c *CapabilityControls) Acknowledge(k RoutineScope, r *CapabilityReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) && !c.state.Busy && r != nil && c.state.Review != nil && r.Sequence == c.state.Review.Sequence {
		c.state.Review.Acknowledged = ack
		c.generation++
	}
}
func (c *CapabilityControls) BeginSave(k RoutineScope, r *CapabilityReview) (CapabilityTicket, pluginrpc.DshbotCapabilityCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotCapabilityCommand
	actual := c.state.Review
	if !c.editable(k) || r == nil || actual == nil || actual.Sequence != r.Sequence || !actual.Acknowledged || !pluginrpc.SameDshbotCapabilities(actual.Command.Expected, c.state.Status) {
		return CapabilityTicket{}, zero, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Loaded = false
	cmd := actual.Command
	cmd.Expected = pluginrpc.CloneDshbotCapabilities(cmd.Expected)
	cmd.Desired = pluginrpc.CloneDshbotCapabilityPolicies(cmd.Desired)
	return CapabilityTicket{k, c.generation}, cmd, true
}
func (c *CapabilityControls) FinishSave(t CapabilityTicket, out pluginrpc.DshbotCapabilityReceipt, err error) bool {
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
		c.state.Error = "保存结果未知；草稿锁定，先读回核实，不自动重试 / Outcome uncertain; read back before saving again"
		if !out.Attempted {
			c.state.Error = "发送前资料或权限已变化，未保存；草稿保留，请刷新重新审阅 / Preflight changed; no save sent"
		}
		return true
	}
	c.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	c.state.Error = ""
	c.state.Notice = "Host 已返回保存回执；草稿保留，刷新核实实际配置与警告 / Saved receipt; read back to reconcile"
	return true
}

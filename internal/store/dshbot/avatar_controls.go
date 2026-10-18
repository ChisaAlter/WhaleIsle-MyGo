package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type AvatarTicket struct {
	Scope                RoutineScope
	Generation, Sequence uint64
}
type AvatarConfirmation struct {
	Command      pluginrpc.DshbotAvatarCommand
	Sequence     uint64
	Acknowledged bool
}
type AvatarSnapshot struct {
	ImageImport                                         AvatarImageImportSnapshot
	Scope                                               RoutineScope
	Generation                                          uint64
	Loaded, Busy, Writable, Dirty, Uncertain, Candidate bool
	Draft                                               pluginrpc.DshbotAvatarValues
	Status                                              pluginrpc.DshbotCapabilityStatus
	Review                                              *AvatarConfirmation
	Warnings                                            []string
	Error, Notice                                       string
}
type avatarAttempt struct {
	Command  pluginrpc.DshbotAvatarCommand
	Sequence uint64
	Sent     bool
}
type avatarEntry struct {
	state      AvatarSnapshot
	attempt    *avatarAttempt
	loading    uint64
	hasDraft   bool
	imageDraft bool
}

// Scope-local entries preserve sent requests across hidden/reopened frames. No
// old callback can dispatch, adopt or write status into a replacement account.
type AvatarControls struct {
	mu                   sync.Mutex
	active               RoutineScope
	generation, sequence uint64
	entries              map[RoutineScope]*avatarEntry
	imageImport          *avatarImagePending
}

func (c *AvatarControls) entry(k RoutineScope) *avatarEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*avatarEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &avatarEntry{state: AvatarSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *AvatarControls) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && c.active == k
}
func (c *AvatarControls) Bind(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == k {
		return
	}
	c.clearImageImportLocked()
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
func (c *AvatarControls) Snapshot(k RoutineScope) AvatarSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return AvatarSnapshot{Scope: k, Generation: c.generation}
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotCapabilities(v.Status)
	v.Warnings = append([]string(nil), v.Warnings...)
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotAvatarCommand(r.Command)
		v.Review = &r
	}
	return v
}
func (c *AvatarControls) ViewCurrent(k RoutineScope, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && c.generation == g
}
func (c *AvatarControls) BeginLoad(k RoutineScope) (AvatarTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return AvatarTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy || e.state.ImageImport.Busy || e.state.ImageImport.Ready {
		return AvatarTicket{}, false
	}
	c.sequence++
	c.generation++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = ""
	return AvatarTicket{k, c.generation, c.sequence}, true
}
func (c *AvatarControls) LoadCurrent(t AvatarTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.loading == t.Sequence
}
func (c *AvatarControls) FinishLoad(t AvatarTicket, status pluginrpc.DshbotCapabilityStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	c.generation++
	value, ve := pluginrpc.DshbotAvatarValue(status)
	if err != nil || ve != nil || status.Bot.ID != t.Scope.BotID {
		e.state.Error = "头像未核实；保留草稿，请刷新 / Read unavailable; draft retained"
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(status)
	e.state.Loaded = true
	e.state.Writable = writable
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if pluginrpc.DshbotAvatarApplied(e.attempt.Command, status) == nil {
			e.state.Candidate = true
			e.state.Notice = "完整资料已匹配审阅值，核对完整头像后可采用读回 / Matching profile found; inspect and adopt"
		} else {
			e.state.Notice = "结果仍未知；仅保留读回，不允许重发。请在 Bot Web 核对 / Unknown outcome remains locked"
		}
		return true
	}
	draftValue, draftErr := pluginrpc.ParseDshbotAvatar(e.state.Draft)
	if !e.hasDraft || !e.state.Dirty || draftErr == nil && draftValue == value {
		e.state.Draft = value
	}
	e.hasDraft = true
	e.state.Dirty = e.state.Draft != value
	e.state.Notice = "已读回；剩余草稿需要重新审阅 / Read back; review remaining changes"
	return true
}
func (c *AvatarControls) editable(k RoutineScope) *avatarEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if !e.state.Loaded || !e.state.Writable || e.state.Busy || e.attempt != nil || e.state.Status.WriteError != "" || e.state.ImageImport.Busy || e.state.ImageImport.Ready {
		return nil
	}
	return e
}
func (c *AvatarControls) SetDraft(k RoutineScope, v pluginrpc.DshbotAvatarValues) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || e.state.Review != nil {
		return
	}
	before, err := pluginrpc.DshbotAvatarValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = v
	e.state.Dirty = v != before
	e.imageDraft = e.state.Dirty && v.Kind == "image"
	e.hasDraft = true
	c.generation++
}
func (c *AvatarControls) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	before, err := pluginrpc.DshbotAvatarValue(e.state.Status)
	if err != nil {
		return
	}
	e.state.Draft = before
	e.imageDraft = false
	e.state.Dirty = false
	e.state.Review = nil
	e.state.Error = ""
	c.generation++
}
func (c *AvatarControls) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Dirty {
		return
	}
	values, err := pluginrpc.ParseDshbotAvatar(e.state.Draft)
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	cmd := pluginrpc.DshbotAvatarCommand{Expected: pluginrpc.CloneDshbotCapabilities(e.state.Status), Values: values}
	if err := pluginrpc.ValidateDshbotAvatarCommand(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	c.generation++
	e.state.Review = &AvatarConfirmation{Command: cmd, Sequence: c.sequence}
	e.state.Error = ""
}
func (c *AvatarControls) Dismiss(k RoutineScope, r *AvatarConfirmation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if !e.state.Busy && r != nil && e.state.Review != nil && r.Sequence == e.state.Review.Sequence {
		e.state.Review = nil
		if e.imageDraft {
			if before, err := pluginrpc.DshbotAvatarValue(e.state.Status); err == nil {
				e.state.Draft = before
				e.state.Dirty = false
			}
			e.imageDraft = false
		}
		c.generation++
	}
}
func (c *AvatarControls) Acknowledge(k RoutineScope, r *AvatarConfirmation, ack bool) {
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
func (c *AvatarControls) BeginSave(k RoutineScope, r *AvatarConfirmation) (AvatarTicket, pluginrpc.DshbotAvatarCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero pluginrpc.DshbotAvatarCommand
	e := c.editable(k)
	if e == nil {
		return AvatarTicket{}, zero, false
	}
	values, parseErr := pluginrpc.ParseDshbotAvatar(e.state.Draft)
	if parseErr != nil || r == nil || e.state.Review == nil || e.state.Review.Sequence != r.Sequence || !e.state.Review.Acknowledged || !pluginrpc.SameDshbotCapabilities(e.state.Status, r.Command.Expected) || values != r.Command.Values {
		return AvatarTicket{}, zero, false
	}
	cmd := pluginrpc.CloneDshbotAvatarCommand(e.state.Review.Command)
	c.sequence++
	c.generation++
	e.attempt = &avatarAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Error = ""
	return AvatarTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *AvatarControls) current(t AvatarTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *AvatarControls) Current(t AvatarTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *AvatarControls) MarkSent(t AvatarTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) || c.entries[t.Scope].attempt.Sent {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *AvatarControls) AbortUnsent(t AvatarTicket, err error) {
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
func (c *AvatarControls) FinishSave(t AvatarTicket, out pluginrpc.DshbotCapabilityReceipt, err error) {
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
		err = pluginrpc.DshbotAvatarApplied(e.attempt.Command, out.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "结果未知；保留并锁定草稿，只读回，不自动重试 / Unknown; read back only"
		return
	}
	e.state.Status = pluginrpc.CloneDshbotCapabilities(out.Status)
	e.state.Draft = e.attempt.Command.Values
	e.state.Dirty = false
	e.state.Uncertain = false
	e.state.Candidate = false
	e.state.Error = ""
	e.state.Notice = "Host 已返回完整头像回执；请刷新核对展示、警告与目标状态 / Profile saved; verify presentation, warnings and goal state"
	e.attempt = nil
}
func (c *AvatarControls) Adopt(k RoutineScope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate || pluginrpc.DshbotAvatarApplied(e.attempt.Command, e.state.Status) != nil {
		return false
	}
	e.state.Draft = e.attempt.Command.Values
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

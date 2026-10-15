package dshbot

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoomEditDraft struct {
	MemberBotIDs         []string
	MaxRounds, MaxSpeaks string
	Version              uint64
}
type RoomEditReview struct {
	Command           pluginrpc.DshbotRoomEditCommand
	Version, Sequence uint64
	Acknowledged      bool
}
type RoomEditSnapshot struct {
	Scope                                        string
	Generation                                   uint64
	Loaded, Writable, Busy, Uncertain, Candidate bool
	Status                                       pluginrpc.DshbotRoomEditStatus
	Draft                                        *RoomEditDraft
	Review                                       *RoomEditReview
	Error, Notice                                string
}
type RoomEditTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type roomEditAttempt struct {
	Command  pluginrpc.DshbotRoomEditCommand
	Sequence uint64
	Sent     bool
}
type roomEditEntry struct {
	state   RoomEditSnapshot
	attempt *roomEditAttempt
	loading uint64
}

// Entries survive navigation in this process. A sent write always settles its
// original account+room, without reviving a retired confirmation or switching UI.
type RoomEditor struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*roomEditEntry
}

func (c *RoomEditor) entry(k string) *roomEditEntry {
	if c.entries == nil {
		c.entries = map[string]*roomEditEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &roomEditEntry{state: RoomEditSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *RoomEditor) valid(k string) bool { return k != "" && c.active == k && c.binding != "" }
func (c *RoomEditor) Bind(k, binding string) {
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
		e.state.Candidate = false
		e.loading = 0
		if e.attempt == nil || e.state.Uncertain {
			e.state.Busy = false
		}
	}
}
func (c *RoomEditor) Suspend() { c.Bind("", "") }
func (c *RoomEditor) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *RoomEditor) Snapshot(k string) RoomEditSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return RoomEditSnapshot{Scope: k, Generation: c.generation}
	}
	v := e.state
	v.Generation = c.generation
	v.Status = pluginrpc.CloneDshbotRoomEdit(v.Status)
	if v.Draft != nil {
		d := *v.Draft
		d.MemberBotIDs = append([]string(nil), d.MemberBotIDs...)
		v.Draft = &d
	}
	if v.Review != nil {
		r := *v.Review
		r.Command = pluginrpc.CloneDshbotRoomEditCommand(r.Command)
		v.Review = &r
	}
	return v
}
func (c *RoomEditor) BeginLoad(k string) (RoomEditTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return RoomEditTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return RoomEditTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = false
	e.state.Error = ""
	return RoomEditTicket{k, c.generation, c.sequence}, true
}
func (c *RoomEditor) FinishLoad(t RoomEditTicket, status pluginrpc.DshbotRoomEditStatus, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || t.Generation != c.generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	e.state.Writable = writable
	if err != nil {
		e.state.Error = err.Error()
		return true
	}
	e.state.Status = pluginrpc.CloneDshbotRoomEdit(status)
	e.state.Loaded = true
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		if match := pluginrpc.DshbotRoomEditApplied(e.attempt.Command, status); match == nil {
			e.state.Candidate = true
			e.state.Notice = "读回已匹配审阅后的完整资料；核对后采用 / Exact reviewed profile found; inspect and adopt"
		} else {
			e.state.Notice = "读回尚未完整匹配；草稿仍锁定，不重发。可通过 Bot Web 核对 / Not confirmed; no retry"
			e.state.Error = match.Error()
		}
	}
	return true
}
func (c *RoomEditor) Start(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.state.Status.WriteError != "" || e.attempt != nil || e.state.Draft != nil {
		return
	}
	r := e.state.Status.Item
	e.state.Draft = &RoomEditDraft{MemberBotIDs: append([]string(nil), r.MemberBotIDs...), MaxRounds: strconv.Itoa(r.MaxRounds), MaxSpeaks: strconv.Itoa(r.MaxSpeaks)}
	e.state.Error = ""
	e.state.Notice = ""
}
func (c *RoomEditor) editable(k string) *roomEditEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft == nil {
		return nil
	}
	return e
}
func (c *RoomEditor) Change(k, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	switch field {
	case "maxRounds":
		e.state.Draft.MaxRounds = value
	case "maxSpeaks":
		e.state.Draft.MaxSpeaks = value
	default:
		return
	}
	e.state.Draft.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *RoomEditor) ToggleMember(k, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded {
		return
	}
	ids := e.state.Draft.MemberBotIDs
	found := -1
	for i, v := range ids {
		if v == id {
			found = i
			break
		}
	}
	if found >= 0 {
		ids = append(ids[:found], ids[found+1:]...)
	} else {
		available := false
		for _, row := range e.state.Status.Catalog.Items {
			if row.ID == id && row.Kind == "bot" {
				available = true
			}
		}
		if !available {
			return
		}
		ids = append(ids, id)
	}
	e.state.Draft.MemberBotIDs = ids
	e.state.Draft.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *RoomEditor) Discard(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.editable(k); e != nil {
		e.state.Draft = nil
		e.state.Review = nil
		e.state.Error = ""
	}
}
func (c *RoomEditor) Dismiss(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *RoomEditor) Prepare(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable {
		return
	}
	e.state.Review = nil
	d := e.state.Draft
	r, er := strconv.Atoi(strings.TrimSpace(d.MaxRounds))
	s, es := strconv.Atoi(strings.TrimSpace(d.MaxSpeaks))
	if er != nil || es != nil {
		e.state.Error = "轮数和发言上限须为整数 / Limits must be integers"
		return
	}
	cmd := pluginrpc.DshbotRoomEditCommand{Expected: pluginrpc.CloneDshbotRoomEdit(e.state.Status), Values: pluginrpc.DshbotRoomEditValues{MemberBotIDs: append([]string(nil), d.MemberBotIDs...), MaxRounds: r, MaxSpeaks: s}}
	if err := pluginrpc.ValidateDshbotRoomEdit(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	before := cmd.Expected.Item
	if r == before.MaxRounds && s == before.MaxSpeaks && reflect.DeepEqual(d.MemberBotIDs, before.MemberBotIDs) {
		e.state.Error = "草稿没有变更 / No changes to save"
		return
	}
	c.sequence++
	e.state.Review = &RoomEditReview{Command: cmd, Version: d.Version, Sequence: c.sequence}
	e.state.Error = ""
}
func sameRoomEditReview(a, b *RoomEditReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && a.Version == b.Version && reflect.DeepEqual(a.Command, b.Command)
}
func (c *RoomEditor) Acknowledge(k string, r *RoomEditReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameRoomEditReview(e.state.Review, r) {
			e.state.Review.Acknowledged = ack
		}
	}
}
func (c *RoomEditor) Begin(k string, r *RoomEditReview) (RoomEditTicket, pluginrpc.DshbotRoomEditCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotRoomEditCommand
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable || !sameRoomEditReview(e.state.Review, r) || !e.state.Review.Acknowledged || e.state.Draft.Version != r.Version || !pluginrpc.SameDshbotRoomEdit(e.state.Status, r.Command.Expected) {
		return RoomEditTicket{}, empty, false
	}
	cmd := pluginrpc.CloneDshbotRoomEditCommand(r.Command)
	c.sequence++
	e.attempt = &roomEditAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return RoomEditTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *RoomEditor) current(t RoomEditTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *RoomEditor) Current(t RoomEditTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *RoomEditor) MarkSent(t RoomEditTicket) bool {
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
func (c *RoomEditor) AbortUnsent(t RoomEditTicket, err error) {
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
	e.state.Notice = "未发送保存，草稿保留；刷新后重新审阅 / Not sent; draft retained"
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *RoomEditor) Finish(t RoomEditTicket, result pluginrpc.DshbotRoomEditResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	if !result.Attempted {
		e.attempt = nil
		e.state.Uncertain = false
		e.state.Notice = "本地校验拒绝，未发送；草稿保留 / Not sent; draft retained"
		if err != nil {
			e.state.Error = err.Error()
		}
		return
	}
	if err == nil {
		err = pluginrpc.DshbotRoomEditApplied(e.attempt.Command, result.Status)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "保存结果未知；保留并锁定草稿，仅可读回，不自动重试 / Unknown outcome; read back only"
		return
	}
	e.state.Status = pluginrpc.CloneDshbotRoomEdit(result.Status)
	e.state.Draft = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "Host 已确认群聊资料保存 / Host confirmed group update"
	if len(result.Warnings) > 0 {
		e.state.Notice += "\n" + strings.Join(result.Warnings, "\n")
	}
	e.attempt = nil
}
func (c *RoomEditor) Adopt(k string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || !e.state.Candidate {
		return false
	}
	if pluginrpc.DshbotRoomEditApplied(e.attempt.Command, e.state.Status) != nil {
		e.state.Error = fmt.Sprint("readback no longer matches")
		return false
	}
	e.state.Candidate = false
	e.state.Draft = nil
	e.state.Review = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "已采用读回的完整匹配资料；没有重发更新。会话展示副作用不能仅凭目录核实 / Adopted matching catalog; presentation side effects remain unverified"
	e.attempt = nil
	return true
}

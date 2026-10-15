package dshbot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoomCreateDraft struct {
	ID, SessionID, Name, Description, WorkspaceID, ScratchCwd, MaxRounds, MaxSpeaks string
	MemberBotIDs                                                                    []string
	Version                                                                         uint64
}
type RoomCreateReview struct {
	Command           pluginrpc.DshbotRoomCreateCommand
	Version, Sequence uint64
	Acknowledged      bool
}
type RoomCreateSnapshot struct {
	CreatedScratchCwd                 string
	Scope                             string
	Generation                        uint64
	Loaded, Writable, Busy, Uncertain bool
	Catalog                           pluginrpc.DshbotRoomCatalog
	Draft                             *RoomCreateDraft
	Review                            *RoomCreateReview
	Candidate                         *pluginrpc.DshbotRoomItem
	Created                           pluginrpc.DshbotRoomItem
	Error, Notice                     string
}
type RoomCreateTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type roomCreateAttempt struct {
	Command  pluginrpc.DshbotRoomCreateCommand
	Sequence uint64
	Sent     bool
}
type roomCreateEntry struct {
	state   RoomCreateSnapshot
	attempt *roomCreateAttempt
	loading uint64
}

// Account-isolated, memory-only drafts survive closing the panel. Neither an
// empty readback nor a later catalog revision proves a session rollback safe.
// Unknown attempts can only be adopted by exact ID; never automatically resent.
type RoomCreator struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*roomCreateEntry
}

func (c *RoomCreator) entry(k string) *roomCreateEntry {
	if c.entries == nil {
		c.entries = map[string]*roomCreateEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &roomCreateEntry{state: RoomCreateSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *RoomCreator) valid(k string) bool { return k != "" && c.active == k && c.binding != "" }
func (c *RoomCreator) Bind(k, binding string) {
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
		if e.attempt == nil {
			e.state.Busy = false
		}
	}
}
func (c *RoomCreator) Suspend() { c.Bind("", "") }
func (c *RoomCreator) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *RoomCreator) Snapshot(k string) RoomCreateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return RoomCreateSnapshot{Scope: k, Generation: c.generation}
	}
	out := e.state
	out.Generation = c.generation
	out.Catalog = pluginrpc.CloneDshbotRoomCatalog(out.Catalog)
	if out.Draft != nil {
		d := *out.Draft
		d.MemberBotIDs = append([]string(nil), d.MemberBotIDs...)
		out.Draft = &d
	}
	if out.Review != nil {
		r := *out.Review
		r.Command = pluginrpc.CloneDshbotRoomCommand(r.Command)
		out.Review = &r
	}
	if out.Candidate != nil {
		r := *out.Candidate
		r.MemberBotIDs = append([]string(nil), r.MemberBotIDs...)
		out.Candidate = &r
	}
	out.Created.MemberBotIDs = append([]string(nil), out.Created.MemberBotIDs...)
	return out
}
func (c *RoomCreator) BeginLoad(k string) (RoomCreateTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return RoomCreateTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return RoomCreateTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = nil
	e.state.Error = ""
	return RoomCreateTicket{k, c.generation, c.sequence}, true
}
func (c *RoomCreator) FinishLoad(t RoomCreateTicket, v pluginrpc.DshbotRoomCatalog, writable bool, err error) bool {
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
	e.state.Catalog = pluginrpc.CloneDshbotRoomCatalog(v)
	e.state.Loaded = true
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		row, matchErr := pluginrpc.DshbotCreatedRoom(e.attempt.Command, v)
		if matchErr == nil {
			e.state.Candidate = &row
			e.state.Notice = "目录已有同身份记录；核对后采用，再单独打开 canonical 会话 / Exact catalog record found; inspect and adopt"
		} else {
			e.state.Notice = "读回未确认完整匹配记录。结果仍未知，不解锁重建；可使用 Bot Web 入口核对 / Outcome remains unknown; no retry"
			e.state.Error = matchErr.Error()
		}
	}
	return true
}
func (c *RoomCreator) Start(k, cwd string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.attempt != nil || e.state.Draft != nil {
		return
	}
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		e.state.Error = err.Error()
		return
	}
	id := hex.EncodeToString(bytes[:])
	e.state.Draft = &RoomCreateDraft{ID: "room-" + id, SessionID: "session-room-" + id, Name: "新群聊", ScratchCwd: strings.TrimSpace(cwd), MaxRounds: "3", MaxSpeaks: "10"}
	e.state.Created = pluginrpc.DshbotRoomItem{}
	e.state.CreatedScratchCwd = ""
	e.state.Error = ""
	e.state.Notice = ""
}
func (c *RoomCreator) editable(k string) *roomCreateEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft == nil {
		return nil
	}
	return e
}
func (c *RoomCreator) Change(k, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil {
		return
	}
	d := e.state.Draft
	switch field {
	case "name":
		d.Name = value
	case "description":
		d.Description = value
	case "workspaceId":
		d.WorkspaceID = value
	case "scratchCwd":
		d.ScratchCwd = value
	case "maxRounds":
		d.MaxRounds = value
	case "maxSpeaks":
		d.MaxSpeaks = value
	default:
		return
	}
	d.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *RoomCreator) ToggleMember(k, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded {
		return
	}
	ids := e.state.Draft.MemberBotIDs
	found := -1
	for i, v := range ids {
		if id == v {
			found = i
			break
		}
	}
	if found >= 0 {
		ids = append(ids[:found], ids[found+1:]...)
	} else {
		available := false
		for _, row := range e.state.Catalog.Items {
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
func (c *RoomCreator) Discard(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e != nil {
		e.state.Draft = nil
		e.state.Review = nil
		e.state.Error = ""
	}
}
func (c *RoomCreator) Dismiss(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *RoomCreator) Prepare(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable {
		return
	}
	e.state.Review = nil
	d := e.state.Draft
	rounds, rErr := strconv.Atoi(strings.TrimSpace(d.MaxRounds))
	speaks, sErr := strconv.Atoi(strings.TrimSpace(d.MaxSpeaks))
	if rErr != nil || sErr != nil {
		e.state.Error = "轮数和发言次数须为整数 / Limits must be integers"
		return
	}
	cmd := pluginrpc.DshbotRoomCreateCommand{ID: d.ID, SessionID: d.SessionID, Expected: pluginrpc.CloneDshbotRoomCatalog(e.state.Catalog), Draft: pluginrpc.DshbotRoomDraft{Name: strings.TrimSpace(d.Name), Description: strings.TrimSpace(d.Description), WorkspaceID: strings.TrimSpace(d.WorkspaceID), ScratchCwd: strings.TrimSpace(d.ScratchCwd), MemberBotIDs: append([]string(nil), d.MemberBotIDs...), MaxRounds: rounds, MaxSpeaks: speaks}}
	if err := pluginrpc.ValidateDshbotRoomCreate(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	e.state.Review = &RoomCreateReview{Command: cmd, Version: d.Version, Sequence: c.sequence}
	e.state.Error = ""
}
func sameRoomReview(a, b *RoomCreateReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && a.Version == b.Version && reflect.DeepEqual(a.Command, b.Command)
}
func (c *RoomCreator) Acknowledge(k string, r *RoomCreateReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameRoomReview(e.state.Review, r) {
			e.state.Review.Acknowledged = ack
		}
	}
}
func (c *RoomCreator) Begin(k string, r *RoomCreateReview) (RoomCreateTicket, pluginrpc.DshbotRoomCreateCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotRoomCreateCommand
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable || !sameRoomReview(e.state.Review, r) || !e.state.Review.Acknowledged || e.state.Draft.Version != r.Version || !pluginrpc.SameDshbotRoomCatalog(e.state.Catalog, r.Command.Expected) {
		return RoomCreateTicket{}, empty, false
	}
	cmd := pluginrpc.CloneDshbotRoomCommand(r.Command)
	c.sequence++
	e.attempt = &roomCreateAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return RoomCreateTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *RoomCreator) current(t RoomCreateTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *RoomCreator) Current(t RoomCreateTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *RoomCreator) MarkSent(t RoomCreateTicket) bool {
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
func (c *RoomCreator) AbortUnsent(t RoomCreateTicket, err error) {
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
	e.state.Notice = "未发送创建；草稿保留，刷新后重新审阅 / Not sent; draft retained"
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *RoomCreator) Finish(t RoomCreateTicket, result pluginrpc.DshbotRoomCreateResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	if err != nil && result.Rejected {
		e.attempt = nil
		e.state.Uncertain = false
		e.state.Error = err.Error()
		e.state.Notice = "写入前拒绝；保留草稿，请刷新后重新审阅 / Rejected before write; draft retained"
		return
	}
	if err == nil {
		row, validationErr := pluginrpc.DshbotCreatedRoom(e.attempt.Command, result.Catalog)
		err = validationErr
		if err == nil && !reflect.DeepEqual(row, result.Item) {
			err = fmt.Errorf("inconsistent room receipt")
		}
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "创建结果未知，草稿与身份已锁定。仅可读回，不重复创建 / Unknown outcome; read back without retry"
		return
	}
	e.state.Created = result.Item
	e.state.CreatedScratchCwd = e.attempt.Command.Draft.ScratchCwd
	e.state.Catalog = pluginrpc.CloneDshbotRoomCatalog(result.Catalog)
	e.state.Draft = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "Host 已确认群聊和专属会话创建。打开失败不代表创建失败 / Room created; opening failure is separate"
	e.attempt = nil
}
func (c *RoomCreator) Adopt(k, id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || e.state.Candidate == nil || e.state.Candidate.ID != id {
		return false
	}
	e.state.Created = *e.state.Candidate
	e.state.CreatedScratchCwd = e.attempt.Command.Draft.ScratchCwd
	e.state.Candidate = nil
	e.state.Draft = nil
	e.state.Review = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "已采用目录中的同身份群聊，未重发创建。打开会核实或恢复专属会话 / Adopted existing room; opening verifies/restores its canonical session"
	e.attempt = nil
	return true
}

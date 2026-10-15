package dshbot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type BotCreateDraft struct {
	ID, SessionID, Name, Title, Description, WorkspaceID, ScratchCwd string
	Provider, Model, ReasoningEffort                                 string
	MemoryReview, Notify                                             bool
	Version                                                          uint64
}
type BotCreateReview struct {
	Command           pluginrpc.DshbotBotCreateCommand
	Version, Sequence uint64
	Acknowledged      bool
}
type BotCreateSnapshot struct {
	CreatedScratchCwd                 string
	Scope                             string
	Generation                        uint64
	Loaded, Writable, Busy, Uncertain bool
	Catalog                           pluginrpc.DshbotBotCatalog
	Draft                             *BotCreateDraft
	Review                            *BotCreateReview
	Pending                           *pluginrpc.DshbotBotCreateCommand
	Candidate                         *pluginrpc.DshbotBotItem
	Created                           pluginrpc.DshbotBotItem
	Error, Notice                     string
}
type BotCreateTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type botCreateAttempt struct {
	Command  pluginrpc.DshbotBotCreateCommand
	Sequence uint64
	Sent     bool
}
type botCreateEntry struct {
	state   BotCreateSnapshot
	attempt *botCreateAttempt
	loading uint64
}

// Account-isolated, memory-only drafts survive closing the panel. Neither an
// empty readback nor a later catalog revision proves a session rollback safe.
// Unknown attempts can only be adopted by exact ID; never automatically resent.
type BotCreator struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*botCreateEntry
}

func (c *BotCreator) entry(k string) *botCreateEntry {
	if c.entries == nil {
		c.entries = map[string]*botCreateEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &botCreateEntry{state: BotCreateSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *BotCreator) valid(k string) bool { return k != "" && c.active == k && c.binding != "" }
func (c *BotCreator) Bind(k, binding string) {
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
func (c *BotCreator) Suspend() { c.Bind("", "") }
func (c *BotCreator) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *BotCreator) Snapshot(k string) BotCreateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return BotCreateSnapshot{Scope: k, Generation: c.generation}
	}
	out := e.state
	if e.attempt != nil {
		pending := pluginrpc.CloneDshbotBotCommand(e.attempt.Command)
		out.Pending = &pending
	}
	out.Generation = c.generation
	out.Catalog = pluginrpc.CloneDshbotBotCatalog(out.Catalog)
	if out.Draft != nil {
		d := *out.Draft
		out.Draft = &d
	}
	if out.Review != nil {
		r := *out.Review
		r.Command = pluginrpc.CloneDshbotBotCommand(r.Command)
		out.Review = &r
	}
	if out.Candidate != nil {
		r := *out.Candidate
		r = pluginrpc.CloneDshbotBotItem(r)
		out.Candidate = &r
	}
	out.Created = pluginrpc.CloneDshbotBotItem(out.Created)
	return out
}
func (c *BotCreator) BeginLoad(k string) (BotCreateTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return BotCreateTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return BotCreateTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidate = nil
	e.state.Error = ""
	return BotCreateTicket{k, c.generation, c.sequence}, true
}
func (c *BotCreator) FinishLoad(t BotCreateTicket, v pluginrpc.DshbotBotCatalog, writable bool, err error) bool {
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
	e.state.Catalog = pluginrpc.CloneDshbotBotCatalog(v)
	e.state.Loaded = true
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		row, matchErr := pluginrpc.DshbotCreatedBot(e.attempt.Command, v)
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
func (c *BotCreator) Start(k, cwd string) {
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
	e.state.Draft = &BotCreateDraft{ID: "bot-" + id, SessionID: "session-bot-" + id, Name: "新 Bot", ScratchCwd: strings.TrimSpace(cwd), MemoryReview: true, Notify: true}
	e.state.Created = pluginrpc.DshbotBotItem{}
	e.state.CreatedScratchCwd = ""
	e.state.Error = ""
	e.state.Notice = ""
}
func (c *BotCreator) editable(k string) *botCreateEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft == nil {
		return nil
	}
	return e
}
func (c *BotCreator) Change(k, field, value string) {
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
	case "title":
		d.Title = value
	case "description":
		d.Description = value
	case "workspaceId":
		d.WorkspaceID = value
	case "scratchCwd":
		d.ScratchCwd = value
	case "provider":
		d.Provider = value
	case "model":
		d.Model = value
	case "reasoningEffort":
		d.ReasoningEffort = value
	case "memoryReview":
		if value != "true" && value != "false" {
			return
		}
		d.MemoryReview = value == "true"
	case "notify":
		if value != "true" && value != "false" {
			return
		}
		d.Notify = value == "true"
	default:
		return
	}
	d.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *BotCreator) Discard(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e != nil {
		e.state.Draft = nil
		e.state.Review = nil
		e.state.Error = ""
	}
}
func (c *BotCreator) Dismiss(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *BotCreator) Prepare(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable {
		return
	}
	e.state.Review = nil
	d := e.state.Draft
	cmd := pluginrpc.DshbotBotCreateCommand{ID: d.ID, SessionID: d.SessionID, Expected: pluginrpc.CloneDshbotBotCatalog(e.state.Catalog), Draft: pluginrpc.DshbotBotDraft{Name: strings.TrimSpace(d.Name), Title: strings.TrimSpace(d.Title), Description: strings.TrimSpace(d.Description), WorkspaceID: strings.TrimSpace(d.WorkspaceID), ScratchCwd: strings.TrimSpace(d.ScratchCwd), Provider: strings.TrimSpace(d.Provider), Model: strings.TrimSpace(d.Model), ReasoningEffort: strings.TrimSpace(d.ReasoningEffort), MemoryReview: d.MemoryReview, Notify: d.Notify}}
	if err := pluginrpc.ValidateDshbotBotCreate(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	e.state.Review = &BotCreateReview{Command: cmd, Version: d.Version, Sequence: c.sequence}
	e.state.Error = ""
}
func sameBotReview(a, b *BotCreateReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && a.Version == b.Version && reflect.DeepEqual(a.Command, b.Command)
}
func (c *BotCreator) Acknowledge(k string, r *BotCreateReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameBotReview(e.state.Review, r) {
			e.state.Review.Acknowledged = ack
		}
	}
}
func (c *BotCreator) Begin(k string, r *BotCreateReview) (BotCreateTicket, pluginrpc.DshbotBotCreateCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotBotCreateCommand
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable || !sameBotReview(e.state.Review, r) || !e.state.Review.Acknowledged || e.state.Draft.Version != r.Version || !pluginrpc.SameDshbotBotCatalog(e.state.Catalog, r.Command.Expected) {
		return BotCreateTicket{}, empty, false
	}
	cmd := pluginrpc.CloneDshbotBotCommand(r.Command)
	c.sequence++
	e.attempt = &botCreateAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return BotCreateTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *BotCreator) current(t BotCreateTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *BotCreator) Current(t BotCreateTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *BotCreator) MarkSent(t BotCreateTicket) bool {
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
func (c *BotCreator) AbortUnsent(t BotCreateTicket, err error) {
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
func (c *BotCreator) Finish(t BotCreateTicket, result pluginrpc.DshbotBotCreateResult, err error) {
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
		row, validationErr := pluginrpc.DshbotCreatedBot(e.attempt.Command, result.Catalog)
		err = validationErr
		if err == nil && !reflect.DeepEqual(row, result.Item) {
			err = fmt.Errorf("inconsistent bot receipt")
		}
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "创建结果未知，草稿与身份已锁定。仅可读回，不重复创建 / Unknown outcome; read back without retry"
		return
	}
	e.state.Created = pluginrpc.CloneDshbotBotItem(result.Item)
	e.state.CreatedScratchCwd = e.attempt.Command.Draft.ScratchCwd
	e.state.Catalog = pluginrpc.CloneDshbotBotCatalog(result.Catalog)
	e.state.Draft = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "Host 已确认Bot和专属会话创建。打开失败不代表创建失败 / Bot created; opening failure is separate"
	e.attempt = nil
}
func (c *BotCreator) Adopt(k, id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil || !e.state.Uncertain || e.state.Candidate == nil || e.state.Candidate.ID != id {
		return false
	}
	e.state.Created = pluginrpc.CloneDshbotBotItem(*e.state.Candidate)
	e.state.CreatedScratchCwd = e.attempt.Command.Draft.ScratchCwd
	e.state.Candidate = nil
	e.state.Draft = nil
	e.state.Review = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "已采用目录中的同身份Bot，未重发创建。打开会核实或恢复专属会话 / Adopted existing bot; opening verifies/restores its canonical session"
	e.attempt = nil
	return true
}

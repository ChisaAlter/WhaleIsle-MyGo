package dshbot

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type BotDuplicateDraft struct {
	SourceID, Name, ScratchCwd string
	Version                    uint64
}
type BotDuplicateReview struct {
	Command               pluginrpc.DshbotBotDuplicateCommand
	Version, Sequence     uint64
	Acknowledged          bool
	ExecutionAcknowledged bool
}
type BotDuplicateSnapshot struct {
	CreatedScratchCwd                 string
	Scope                             string
	Generation                        uint64
	Loaded, Writable, Busy, Uncertain bool
	Catalog                           pluginrpc.DshbotBotCatalog
	Draft                             *BotDuplicateDraft
	Review                            *BotDuplicateReview
	Pending                           *pluginrpc.DshbotBotDuplicateCommand
	Candidates                        []pluginrpc.DshbotBotItem
	Warnings                          []string
	Created                           pluginrpc.DshbotBotItem
	Error, Notice                     string
}
type BotDuplicateTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type botDuplicateAttempt struct {
	Command  pluginrpc.DshbotBotDuplicateCommand
	Sequence uint64
	Sent     bool
}
type botDuplicateEntry struct {
	state   BotDuplicateSnapshot
	attempt *botDuplicateAttempt
	loading uint64
}

// Account-isolated, memory-only drafts survive closing the panel. Neither an
// empty readback nor a later catalog revision proves a session rollback safe.
// Host-generated IDs cannot be causally adopted from readback candidates.
// Only an original successful receipt settles an attempt; never auto-retry.
type BotDuplicator struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*botDuplicateEntry
}

func (c *BotDuplicator) entry(k string) *botDuplicateEntry {
	if c.entries == nil {
		c.entries = map[string]*botDuplicateEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &botDuplicateEntry{state: BotDuplicateSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *BotDuplicator) valid(k string) bool { return k != "" && c.active == k && c.binding != "" }
func (c *BotDuplicator) Bind(k, binding string) {
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
func (c *BotDuplicator) Suspend() { c.Bind("", "") }
func (c *BotDuplicator) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(k) && g == c.generation
}
func (c *BotDuplicator) Snapshot(k string) BotDuplicateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return BotDuplicateSnapshot{Scope: k, Generation: c.generation}
	}
	out := e.state
	if e.attempt != nil {
		pending := pluginrpc.CloneDshbotBotDuplicateCommand(e.attempt.Command)
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
		r.Command = pluginrpc.CloneDshbotBotDuplicateCommand(r.Command)
		out.Review = &r
	}
	out.Candidates = make([]pluginrpc.DshbotBotItem, len(e.state.Candidates))
	for i, row := range e.state.Candidates {
		out.Candidates[i] = pluginrpc.CloneDshbotBotItem(row)
	}
	out.Warnings = append([]string(nil), e.state.Warnings...)
	out.Created = pluginrpc.CloneDshbotBotItem(out.Created)
	return out
}
func (c *BotDuplicator) BeginLoad(k string) (BotDuplicateTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return BotDuplicateTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return BotDuplicateTicket{}, false
	}
	c.sequence++
	e.loading = c.sequence
	e.state.Busy = true
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Candidates = nil
	e.state.Error = ""
	return BotDuplicateTicket{k, c.generation, c.sequence}, true
}
func (c *BotDuplicator) FinishLoad(t BotDuplicateTicket, v pluginrpc.DshbotBotCatalog, writable bool, err error) bool {
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
		e.state.Candidates = pluginrpc.DshbotBotDuplicateCandidates(e.attempt.Command, v)
		e.state.Notice = "仅列出新出现且配置相符的目录候选；不能证明是此请求创建，原 attempt 保持未知且不可重发。Bot 记忆复制结果也未知 / Candidates do not establish causality; no retry"
	}

	return true
}
func (c *BotDuplicator) Start(k, sourceID, cwd string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.attempt != nil || e.state.Draft != nil {
		return
	}
	if _, err := pluginrpc.DshbotBotDuplicateSource(pluginrpc.DshbotBotDuplicateCommand{SourceID: sourceID, Expected: e.state.Catalog}); err != nil {
		e.state.Error = err.Error()
		return
	}
	e.state.Draft = &BotDuplicateDraft{SourceID: sourceID, ScratchCwd: strings.TrimSpace(cwd)}
	e.state.Warnings = nil
	e.state.Created = pluginrpc.DshbotBotItem{}
	e.state.CreatedScratchCwd = ""
	e.state.Error = ""
	e.state.Notice = ""
}
func (c *BotDuplicator) editable(k string) *botDuplicateEntry {
	if !c.valid(k) {
		return nil
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft == nil {
		return nil
	}
	return e
}
func (c *BotDuplicator) Change(k, field, value string) {
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
	case "scratchCwd":
		d.ScratchCwd = value
	default:
		return
	}
	d.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *BotDuplicator) Discard(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e != nil {
		e.state.Draft = nil
		e.state.Review = nil
		e.state.Error = ""
	}
}
func (c *BotDuplicator) Dismiss(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *BotDuplicator) Prepare(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable {
		return
	}
	e.state.Review = nil
	d := e.state.Draft
	cmd := pluginrpc.DshbotBotDuplicateCommand{SourceID: d.SourceID, Name: strings.TrimSpace(d.Name), ScratchCwd: strings.TrimSpace(d.ScratchCwd), Expected: pluginrpc.CloneDshbotBotCatalog(e.state.Catalog)}
	if err := pluginrpc.ValidateDshbotBotDuplicate(cmd); err != nil {
		e.state.Error = err.Error()
		return
	}
	c.sequence++
	e.state.Review = &BotDuplicateReview{Command: cmd, Version: d.Version, Sequence: c.sequence}
	e.state.Error = ""
}
func sameBotReviewForDuplicate(a, b *BotDuplicateReview) bool {
	return a != nil && b != nil && a.Sequence == b.Sequence && a.Version == b.Version && reflect.DeepEqual(a.Command, b.Command)
}
func (c *BotDuplicator) Acknowledge(k string, r *BotDuplicateReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameBotReviewForDuplicate(e.state.Review, r) {
			e.state.Review.Acknowledged = ack
		}
	}
}
func (c *BotDuplicator) AcknowledgeExecution(k string, r *BotDuplicateReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && sameBotReviewForDuplicate(e.state.Review, r) {
			e.state.Review.ExecutionAcknowledged = ack
		}
	}
}
func (c *BotDuplicator) Begin(k string, r *BotDuplicateReview) (BotDuplicateTicket, pluginrpc.DshbotBotDuplicateCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotBotDuplicateCommand
	e := c.editable(k)
	if e == nil || !e.state.Loaded || !e.state.Writable || !sameBotReviewForDuplicate(e.state.Review, r) || !e.state.Review.Acknowledged || (pluginrpc.DshbotBotDuplicateHasObjective(r.Command) && !e.state.Review.ExecutionAcknowledged) || e.state.Draft.Version != r.Version || !pluginrpc.SameDshbotBotCatalog(e.state.Catalog, r.Command.Expected) {
		return BotDuplicateTicket{}, empty, false
	}
	cmd := pluginrpc.CloneDshbotBotDuplicateCommand(r.Command)
	cmd.AllowObjectiveExecution = e.state.Review.ExecutionAcknowledged
	c.sequence++
	e.attempt = &botDuplicateAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return BotDuplicateTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *BotDuplicator) current(t BotDuplicateTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.generation == t.Generation && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *BotDuplicator) Current(t BotDuplicateTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *BotDuplicator) MarkSent(t BotDuplicateTicket) bool {
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
func (c *BotDuplicator) AbortUnsent(t BotDuplicateTicket, err error) {
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
func (c *BotDuplicator) Finish(t BotDuplicateTicket, result pluginrpc.DshbotBotDuplicateResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Warnings = append([]string(nil), result.Warnings...)
	if err != nil && result.Rejected {
		e.attempt = nil
		e.state.Uncertain = false
		e.state.Error = err.Error()
		e.state.Notice = "写入前拒绝；保留草稿，请刷新后重新审阅 / Rejected before write; draft retained"
		return
	}
	if err == nil {
		row, validationErr := pluginrpc.DshbotDuplicatedBot(e.attempt.Command, result.Catalog, result.Item.ID, result.Item.SessionID)
		err = validationErr
		if err == nil && !pluginrpc.SameDshbotBotCatalog(pluginrpc.DshbotBotCatalog{Items: []pluginrpc.DshbotBotItem{row}}, pluginrpc.DshbotBotCatalog{Items: []pluginrpc.DshbotBotItem{result.Item}}) {
			err = fmt.Errorf("inconsistent duplicate receipt")
		}
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "复制结果未知，原来源与本次输入已锁定。Host 自生新身份；只能读回候选，不能自动归因或重试 / Unknown duplicate outcome; candidates are not causal receipts"
		return
	}
	e.state.Created = pluginrpc.CloneDshbotBotItem(result.Item)
	e.state.CreatedScratchCwd = e.attempt.Command.ScratchCwd
	e.state.Catalog = pluginrpc.CloneDshbotBotCatalog(result.Catalog)
	e.state.Draft = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "Host 已确认副本资料与新会话创建；记忆复制状态请看完整 warning。不会自动打开，请单独选择 / Profile/session receipt confirmed; inspect warnings before opening"
	e.attempt = nil
}

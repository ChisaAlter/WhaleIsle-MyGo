package dshbot

import (
	"fmt"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
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
	Error, Notice                                string
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
func (c *BotDeleter) Snapshot(k string) BotDeleteSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return BotDeleteSnapshot{Scope: k, Generation: c.generation}
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
	return out
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
	e.state.Error = ""
	return BotDeleteTicket{k, c.generation, c.sequence}, true
}
func (c *BotDeleter) FinishLoad(t BotDeleteTicket, p pluginrpc.DshbotBotDeletePlan, v pluginrpc.DshbotDeleteCatalog, writable bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if !c.valid(t.Scope) || c.generation != t.Generation || e == nil || e.loading != t.Sequence {
		return false
	}
	e.loading = 0
	e.state.Busy = false
	e.state.Writable = writable
	if err != nil {
		e.state.Error = err.Error()
		return true
	}
	e.state.Loaded = true
	e.state.Error = ""
	if e.attempt != nil && e.state.Uncertain {
		e.state.Observation = pluginrpc.DshbotBotDeleteObservation(e.attempt.Command.Plan, v)
		e.state.Notice = "只读核验不能证明停止、目录、任务或 notepad 的最终结果；原删除仍未知，禁止重发 / Readback does not settle the unknown attempt"
		return true
	}
	e.state.Plan = pluginrpc.CloneDshbotBotDeletePlan(p)
	e.state.Warnings = append([]string(nil), p.Preview.Warnings...)
	e.state.Notice = "完整预览已读取；尚未授权删除 / Preview read; deletion not authorized"
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
		e.state.Error = "删除预览已失效，请重新读取 / Deletion preview expired"
		return
	}
	c.sequence++
	e.state.Review = &BotDeleteReview{Plan: p, Sequence: c.sequence}
	e.state.Error = ""
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
		e.state.Notice = "已取消，未发送删除；下次需重新读取完整预览 / Cancelled before dispatch"
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
		e.state.Error = err.Error()
		return BotDeleteTicket{}, empty, false
	}
	c.sequence++
	e.attempt = &botDeleteAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return BotDeleteTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *BotDeleter) current(t BotDeleteTicket) bool {
	e := c.entries[t.Scope]
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
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || e.attempt.Sent {
		return
	}
	e.attempt = nil
	e.state.Busy = false
	e.state.Loaded = false
	e.state.Review = nil
	e.state.Notice = "未发送删除；需重新读取预览并当次确认 / Deletion not sent"
	if err != nil {
		e.state.Error = err.Error()
	}
}
func (c *BotDeleter) Finish(t BotDeleteTicket, result pluginrpc.DshbotBotDeleteResult, err error) {
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
		e.state.Error = err.Error()
		e.state.Notice = "Host 在副作用前明确拒绝；请重新读取预览并确认 / Rejected before effects"
		return
	}
	if err == nil {
		err = pluginrpc.ValidateDshbotBotDeleteReceipt(e.attempt.Command, result)
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "删除结果未知；可能已停止代理、取消任务、改目录或删除 notepad。只能只读核验，禁止自动重试或假定回滚 / Unknown; side effects may already have occurred"
		return
	}
	e.state.Plan = pluginrpc.CloneDshbotBotDeletePlan(e.attempt.Command.Plan)
	e.state.Completed = true
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = fmt.Sprintf("Host 已确认删除目录目标 %s；history/memory 保留。请检查完整 warning，停止/presentation/notepad 可能有部分失败；不是全部可恢复 / Deletion receipt confirmed; inspect warnings", result.Removed)
	e.attempt = nil
}

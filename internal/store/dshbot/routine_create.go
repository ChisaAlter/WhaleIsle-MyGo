package dshbot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoutineCreateDraft struct {
	Name, Prompt, Hour, Minute, Timezone, MaxRuns      string
	Frequency, OnceDelay, IntervalAmount, IntervalUnit string
	Enabled                                            bool
	Revision                                           int
	SessionID, RequestID                               string
	Version                                            uint64
}
type RoutineCreateReview struct {
	Command pluginrpc.DshbotRoutineCreateCommand
	Version uint64
}
type RoutineCreateSnapshot struct {
	Scope                                  RoutineScope
	Loaded, Busy, Uncertain, RetryVerified bool
	Draft                                  *RoutineCreateDraft
	Review                                 *RoutineCreateReview
	Candidates                             []pluginrpc.DshbotRoutine
	CreatedID, Error, Notice               string
}
type RoutineCreateTicket struct {
	Scope, SourceScope RoutineScope
	Binding, Sequence  uint64
}
type routineCreateAttempt struct {
	Command  pluginrpc.DshbotRoutineCreateCommand
	Sequence uint64
	Sent     bool
}
type routineCreateEntry struct {
	state   RoutineCreateSnapshot
	attempt *routineCreateAttempt
}

// Entries are local to an authenticated Host + Bot, including unresolved creates
// after navigation. They are never copied into another account's active panel.
// Pending identities are memory-only; restart recovery is not claimed.
type RoutineCreator struct {
	mu                sync.Mutex
	active            RoutineScope
	source            RoutineSnapshot
	binding, sequence uint64
	entries           map[RoutineScope]*routineCreateEntry
}

func (c *RoutineCreator) entry(k RoutineScope) *routineCreateEntry {
	if c.entries == nil {
		c.entries = map[RoutineScope]*routineCreateEntry{}
	}
	e := c.entries[k]
	if e == nil {
		e = &routineCreateEntry{state: RoutineCreateSnapshot{Scope: k}}
		c.entries[k] = e
	}
	return e
}
func (c *RoutineCreator) valid(k RoutineScope) bool {
	return k.HostKey != "" && k.BotID != "" && k == c.active && c.source.Scope.BotID == k.BotID && c.binding != 0
}
func (c *RoutineCreator) Sync(k RoutineScope, binding uint64, source RoutineSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k.HostKey == "" || k.BotID == "" {
		c.active = RoutineScope{}
		c.binding = 0
		c.source = RoutineSnapshot{}
		return
	}
	e := c.entry(k)
	if c.active != k || c.binding != binding || c.source.Scope != source.Scope || c.source.Revision != source.Revision || c.source.SessionID != source.SessionID {
		e.state.Review = nil
	}
	c.active = k
	c.binding = binding
	c.source = source
	e.state.Loaded = source.Loaded && !source.Busy
	e.state.RetryVerified = false
	e.state.Candidates = nil
	if e.attempt == nil || e.state.Busy || !e.state.Loaded {
		return
	}
	cmd := e.attempt.Command
	if source.SessionID != cmd.SessionID {
		e.state.Error = "Bot 会话已更换。创建结果待核对，不能重发 / Bot session changed; inspect the original creation"
		return
	}
	if source.Revision < cmd.Revision {
		return
	}
	old := map[string]bool{}
	for _, id := range cmd.BeforeIDs {
		old[id] = true
	}
	for _, r := range source.Routines {
		if !old[r.ID] && r.BotID == k.BotID {
			e.state.Candidates = append(e.state.Candidates, cloneRoutine(r))
		}
	}
	// A newer catalog revision fences any late request carrying the old CAS.
	// Any new target-Bot row, including one edited by another client, stays visible
	// for explicit adoption rather than being assumed not to be our creation.
	e.state.RetryVerified = source.Revision > cmd.Revision && len(e.state.Candidates) == 0
}
func cloneCreateCommand(cmd pluginrpc.DshbotRoutineCreateCommand) pluginrpc.DshbotRoutineCreateCommand {
	cmd.BeforeIDs = append([]string(nil), cmd.BeforeIDs...)
	return cmd
}
func (c *RoutineCreator) Snapshot(k RoutineScope) RoutineCreateSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[k]
	if e == nil {
		return RoutineCreateSnapshot{Scope: k}
	}
	out := e.state
	if out.Draft != nil {
		d := *out.Draft
		out.Draft = &d
	}
	if out.Review != nil {
		r := *out.Review
		r.Command = cloneCreateCommand(r.Command)
		out.Review = &r
	}
	out.Candidates = make([]pluginrpc.DshbotRoutine, len(e.state.Candidates))
	for i, r := range e.state.Candidates {
		out.Candidates[i] = cloneRoutine(r)
	}
	return out
}
func createRequestID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "mygo-routine-" + hex.EncodeToString(b[:]), nil
}

// The browser defaults to a known local IANA zone. On native systems exposing
// only "Local" or an ambiguous abbreviation, require explicit input instead.
func RoutineCreateTimezone(local string) string {
	local = strings.TrimSpace(local)
	if local == "UTC" || strings.Contains(local, "/") {
		return local
	}
	return ""
}
func (c *RoutineCreator) Start(k RoutineScope, zone string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.source.Loaded || c.source.Busy {
		return
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft != nil {
		return
	}
	id, err := createRequestID()
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	e.state.Draft = &RoutineCreateDraft{Name: "新定时任务", Frequency: "daily", OnceDelay: "30m", IntervalAmount: "1", IntervalUnit: "h", Hour: "09", Minute: "00", Timezone: RoutineCreateTimezone(zone), MaxRuns: "0", Revision: c.source.Revision, SessionID: c.source.SessionID, RequestID: id}
	e.state.CreatedID = ""
	e.state.Error = ""
	e.state.Notice = ""
}
func (c *RoutineCreator) Change(k RoutineScope, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return
	}
	e := c.entry(k)
	if e.state.Draft == nil || e.state.Busy || e.attempt != nil {
		return
	}
	d := e.state.Draft
	switch field {
	case "name":
		d.Name = value
	case "prompt":
		d.Prompt = value
	case "frequency":
		if value != "daily" && value != "once" && value != "interval" {
			return
		}
		d.Frequency = value
	case "onceDelay":
		d.OnceDelay = value
	case "intervalAmount":
		d.IntervalAmount = value
	case "intervalUnit":
		if value != "m" && value != "h" && value != "d" {
			return
		}
		d.IntervalUnit = value
	case "hour":
		d.Hour = value
	case "minute":
		d.Minute = value
	case "timezone":
		d.Timezone = value
	case "maxRuns":
		d.MaxRuns = value
	case "enabled":
		d.Enabled = value == "true"
	default:
		return
	}
	d.Version++
	e.state.Review = nil
	e.state.Error = ""
}
func (c *RoutineCreator) Discard(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy && e.attempt == nil {
			e.state.Draft = nil
			e.state.Review = nil
			e.state.Error = ""
		}
	}
}
func (c *RoutineCreator) Dismiss(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(k) {
		e := c.entry(k)
		if !e.state.Busy {
			e.state.Review = nil
		}
	}
}
func (c *RoutineCreator) Rebase(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.source.Loaded || c.source.Busy {
		return
	}
	e := c.entry(k)
	if e.state.Busy || e.state.Draft == nil {
		return
	}
	if e.attempt != nil && !e.state.RetryVerified {
		return
	}
	if e.state.Draft.SessionID != c.source.SessionID {
		e.state.Error = "Bot 会话已更换，请先核对原任务 / Bot identity changed"
		return
	}
	id, err := createRequestID()
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	e.state.Draft.RequestID = id
	e.state.Draft.Revision = c.source.Revision
	e.state.Draft.Version++
	e.state.Review = nil
	e.attempt = nil
	e.state.Uncertain = false
	e.state.RetryVerified = false
	e.state.Error = ""
	e.state.Notice = "已采用当前目录基线，请重新审阅保存 / Current catalog adopted; review creation again"
}
func (c *RoutineCreator) Prepare(k RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) || !c.source.Loaded || c.source.Busy {
		return
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt != nil || e.state.Draft == nil {
		return
	}
	e.state.Review = nil
	d := e.state.Draft
	if d.Revision != c.source.Revision || d.SessionID != c.source.SessionID {
		e.state.Error = "目录已变化；先核对当前目录再审阅创建 / Catalog changed; recheck before reviewing"
		return
	}
	schedule, err := routineCreateSchedule(*d)
	if err != nil {
		e.state.Error = err.Error()
		return
	}
	limit, lErr := strconv.Atoi(strings.TrimSpace(d.MaxRuns))
	if lErr != nil {
		e.state.Error = "执行次数上限须为整数 / Run limit must be an integer"
		return
	}
	zone := strings.TrimSpace(d.Timezone)
	if zone == "" || zone == "Local" {
		e.state.Error = "请明确填写 IANA 时区 / Specify the intended IANA timezone"
		return
	}
	if _, err := time.LoadLocation(zone); err != nil {
		e.state.Error = "时区无效 / Invalid IANA timezone: " + zone
		return
	}
	edit := pluginrpc.DshbotRoutineEdit{Name: d.Name, Prompt: d.Prompt, Schedule: schedule, Timezone: d.Timezone, MaxRuns: limit, Enabled: d.Enabled}
	if err := pluginrpc.ValidateDshbotRoutineEdit(edit); err != nil {
		e.state.Error = err.Error()
		return
	}
	cmd := pluginrpc.DshbotRoutineCreateCommand{BotID: k.BotID, SessionID: d.SessionID, RequestID: d.RequestID, Revision: d.Revision, Edit: edit}
	for _, r := range c.source.Routines {
		cmd.BeforeIDs = append(cmd.BeforeIDs, r.ID)
	}
	e.state.Review = &RoutineCreateReview{Command: cmd, Version: d.Version}
	e.state.Error = ""
}
func (c *RoutineCreator) Begin(k RoutineScope, expected *RoutineCreateReview) (RoutineCreateTicket, pluginrpc.DshbotRoutineCreateCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotRoutineCreateCommand
	if !c.valid(k) || !c.source.Loaded || c.source.Busy {
		return RoutineCreateTicket{}, empty, false
	}
	e := c.entry(k)
	r := e.state.Review
	if r == nil || expected == nil || expected.Command.RequestID != r.Command.RequestID || expected.Version != r.Version || expected.Command.Revision != r.Command.Revision || e.state.Draft == nil || e.state.Busy || e.attempt != nil || r.Version != e.state.Draft.Version || r.Command.Revision != c.source.Revision || r.Command.SessionID != c.source.SessionID {
		return RoutineCreateTicket{}, empty, false
	}
	c.sequence++
	cmd := cloneCreateCommand(r.Command)
	e.attempt = &routineCreateAttempt{Command: cmd, Sequence: c.sequence}
	e.state.Busy = true
	e.state.Review = nil
	e.state.Error = ""
	return RoutineCreateTicket{Scope: k, SourceScope: c.source.Scope, Binding: c.binding, Sequence: c.sequence}, cmd, true
}
func (c *RoutineCreator) current(t RoutineCreateTicket) bool {
	e := c.entries[t.Scope]
	return c.valid(t.Scope) && c.binding == t.Binding && c.source.Scope == t.SourceScope && e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && e.state.Busy
}
func (c *RoutineCreator) Current(t RoutineCreateTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *RoutineCreator) MarkSent(t RoutineCreateTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.entries[t.Scope].attempt.Sent = true
	return true
}
func (c *RoutineCreator) AbortUnsent(t RoutineCreateTicket) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e != nil && e.attempt != nil && e.attempt.Sequence == t.Sequence && !e.attempt.Sent {
		e.attempt = nil
		e.state.Busy = false
		e.state.Review = nil
		e.state.Notice = "未发送创建请求；草稿保留，请重新审阅 / Creation not sent; review the retained draft again"
	}
}
func (c *RoutineCreator) Finish(t RoutineCreateTicket, result pluginrpc.DshbotRoutineCreateResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt == nil || e.attempt.Sequence != t.Sequence || !e.attempt.Sent {
		return
	}
	e.state.Busy = false
	if err != nil && result.Rejected {
		e.attempt = nil
		e.state.Uncertain = false
		e.state.Review = nil
		e.state.Error = err.Error()
		e.state.Notice = "Host 在写入前拒绝创建；草稿保留，修正或刷新后重新审阅 / Creation rejected before write; draft retained"
		return
	}
	if err == nil {
		id, validationErr := pluginrpc.DshbotRoutineCreatedID(e.attempt.Command, result.View)
		err = validationErr
		if err == nil && id != result.ID {
			err = fmt.Errorf("creation returned a different identity")
		}
	}
	if err != nil {
		e.state.Uncertain = true
		e.state.Error = err.Error()
		e.state.Notice = "创建结果待核对，已锁定此请求与草稿；刷新只读目录，勿重复保存 / Outcome uncertain; inspect the catalog before another creation"
		return
	}
	e.state.CreatedID = result.ID
	e.state.Draft = nil
	e.state.Review = nil
	e.state.Uncertain = false
	e.state.Error = ""
	e.state.Notice = "Host 已返回新例行任务 / Created routine: " + result.ID
	e.attempt = nil
}
func (c *RoutineCreator) Adopt(k RoutineScope, id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(k) {
		return false
	}
	e := c.entry(k)
	if e.state.Busy || e.attempt == nil {
		return false
	}
	for _, r := range e.state.Candidates {
		if r.ID == id {
			e.state.CreatedID = id
			e.state.Draft = nil
			e.state.Review = nil
			e.state.Uncertain = false
			e.state.Candidates = nil
			e.state.RetryVerified = false
			e.state.Error = ""
			e.state.Notice = "已采用核对后的已有记录，未发送新建请求 / Using the inspected existing record"
			e.attempt = nil
			return true
		}
	}
	return false
}

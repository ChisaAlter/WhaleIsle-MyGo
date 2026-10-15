package dshbot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoutineScope struct{ HostKey, BotID string }
type RoutineTicket struct {
	Scope      RoutineScope
	generation uint64
}
type RoutineDraft struct {
	ID, SessionID string
	Revision      int
	Edit          pluginrpc.DshbotRoutineEdit
	MaxRunsText   string
	Advanced      RoutineAdvancedDraft
	Version       uint64
}
type RoutineReview struct {
	Command                          pluginrpc.DshbotRoutineCommand
	SessionID                        string
	DraftVersion                     uint64
	Sequence                         uint64
	WorkspaceID                      string
	Advanced                         RoutineAdvancedDraft
	WatchAcknowledged                bool
	Name, Prompt, Schedule, Timezone string
}
type RoutineSnapshot struct {
	Scope                          RoutineScope
	Loaded, Busy                   bool
	Revision                       int
	BotName, SessionID, SelectedID string
	WorkspaceID                    string
	Routines                       []pluginrpc.DshbotRoutine
	MissedIDs                      []string
	MissedKnown                    bool
	Draft                          *RoutineDraft
	Review                         *RoutineReview
	Error, Notice                  string
	Trigger                        RoutineTriggerSnapshot
}
type RoutineControls struct {
	Avatar          AvatarControls
	SenderPolicy    SenderPolicyControls
	TextProfile     TextProfileControls
	Route           RouteControls
	ObjectiveConfig ObjectiveConfigControls
	MemoryReview    MemoryReviewControls
	Capabilities    CapabilityControls
	Objective       ObjectiveControls
	Memory          MemoryControls
	mu              sync.Mutex
	state           RoutineSnapshot
	generation      uint64
	binding         uint64
	reviewSequence  uint64
	triggerSequence uint64
	trigger         routineTriggerState
	drafts          map[string]RoutineDraft
}

func (c *RoutineControls) Bind(scope RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != scope {
		c.TextProfile.Bind(scope)
		c.SenderPolicy.Bind(scope)
		c.Avatar.Bind(scope)
		c.Route.Bind(scope)
		c.Memory.Bind(scope)
		c.MemoryReview.Bind(scope)
		c.Capabilities.Bind(scope)
		c.Objective.Bind(scope)
		c.ObjectiveConfig.Bind(scope)
		c.clearRoutineTrigger()
		c.generation++
		c.binding++
		c.state = RoutineSnapshot{Scope: scope}
		c.drafts = nil
	}
}

// Binding is stable across requests, but changes after closing/reopening or
// moving to another Host/Bot. Old model callbacks cannot consume a new review.
func (c *RoutineControls) Binding(scope RoutineScope) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(scope) {
		return 0
	}
	return c.binding
}
func (c *RoutineControls) Bound(scope RoutineScope, binding uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return binding != 0 && c.valid(scope) && c.binding == binding
}

// Current fences dispatch as well as completion. A discarded ticket must never
// be sent merely because the same authenticated Bot was reopened meanwhile.
func (c *RoutineControls) Current(ticket RoutineTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.matches(ticket) && c.state.Busy && c.valid(ticket.Scope)
}
func cloneRoutine(r pluginrpc.DshbotRoutine) pluginrpc.DshbotRoutine {
	r.ContextFrom = append([]string(nil), r.ContextFrom...)
	r.Watch = append([]byte(nil), r.Watch...)
	if r.RunHistory != nil {
		r.RunHistory = append([]pluginrpc.DshbotRoutineRun{}, r.RunHistory...)
	}
	return r
}
func (c *RoutineControls) Snapshot(scope RoutineScope) RoutineSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != scope {
		return RoutineSnapshot{Scope: scope}
	}
	out := c.state
	out.Trigger = c.trigger.meta
	out.Routines = make([]pluginrpc.DshbotRoutine, len(c.state.Routines))
	for i, r := range c.state.Routines {
		out.Routines[i] = cloneRoutine(r)
	}
	out.MissedIDs = append([]string(nil), out.MissedIDs...)
	if d, ok := c.drafts[out.SelectedID]; ok {
		d = cloneRoutineDraft(d)
		out.Draft = &d
	}
	if out.Review != nil {
		r := cloneRoutineReview(*out.Review)
		out.Review = &r
	}
	return out
}
func (c *RoutineControls) valid(s RoutineScope) bool {
	return s == c.state.Scope && s.HostKey != "" && s.BotID != ""
}
func (c *RoutineControls) matches(t RoutineTicket) bool {
	return t.Scope == c.state.Scope && t.generation == c.generation
}
func (c *RoutineControls) BeginLoad(s RoutineScope) (RoutineTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || c.state.Busy {
		return RoutineTicket{}, false
	}
	c.clearRoutineTrigger()
	c.generation++
	c.state.Busy = true
	c.state.Error = ""
	c.state.Review = nil
	return RoutineTicket{s, c.generation}, true
}
func (c *RoutineControls) replace(v *pluginrpc.DshbotRoutineView) error {
	b, ok := v.Bot(c.state.Scope.BotID)
	if !ok {
		return fmt.Errorf("Bot is no longer available; refresh the roster")
	}
	seen := map[string]bool{}
	rows := []pluginrpc.DshbotRoutine{}
	for _, r := range v.Value.Routines {
		if r.ID == "" || seen[r.ID] {
			return fmt.Errorf("Host returned an invalid routine identity")
		}
		seen[r.ID] = true
		if r.BotID == b.ID {
			rows = append(rows, cloneRoutine(r))
		}
	}
	if c.state.SessionID != "" && c.state.SessionID != b.SessionID {
		c.drafts = nil
		c.state.SelectedID = ""
		c.state.Review = nil
	}
	c.state.BotName = b.Name
	if c.state.BotName == "" {
		c.state.BotName = b.ID
	}
	c.state.SessionID = b.SessionID
	c.state.WorkspaceID = b.WorkspaceID
	c.state.Revision = v.Revision
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Name == rows[j].Name {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].Name < rows[j].Name
	})
	c.state.Routines = rows
	c.state.Loaded = true
	return nil
}
func (c *RoutineControls) FinishLoad(t RoutineTicket, v *pluginrpc.DshbotRoutineView, missed []string, missedErr, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.matches(t) {
		return false
	}
	c.state.Busy = false
	if err == nil {
		err = c.replace(v)
	}
	if err != nil {
		c.state.Loaded = false
		c.state.Error = err.Error()
		return true
	}
	c.state.MissedIDs = nil
	for _, id := range missed {
		if _, ok := c.row(id); ok {
			c.state.MissedIDs = append(c.state.MissedIDs, id)
		}
	}
	c.state.MissedKnown = missedErr == nil
	c.state.Error = ""
	c.state.Notice = "例行任务已从当前 Host 刷新 / Routines refreshed"
	if missedErr != nil {
		c.state.Notice += "；错过记录不可用 / Missed-run status unavailable: " + missedErr.Error()
	}
	return true
}
func (c *RoutineControls) row(id string) (pluginrpc.DshbotRoutine, bool) {
	for _, r := range c.state.Routines {
		if r.ID == id {
			return r, true
		}
	}
	return pluginrpc.DshbotRoutine{}, false
}
func (c *RoutineControls) Select(s RoutineScope, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || c.state.Busy {
		return
	}
	if _, ok := c.row(id); ok {
		if id != c.state.SelectedID {
			c.clearRoutineTrigger()
		}
		c.state.SelectedID = id
		c.state.Review = nil
		c.state.Error = ""
	}
}
func (c *RoutineControls) StartEdit(s RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || c.state.Busy || !c.state.Loaded {
		return
	}
	r, ok := c.row(c.state.SelectedID)
	if !ok {
		return
	}
	if c.drafts == nil {
		c.drafts = map[string]RoutineDraft{}
	}
	if _, ok := c.drafts[r.ID]; ok {
		return
	}
	schedule := r.Schedule
	if schedule == "" && r.IntervalMinutes > 0 {
		schedule = fmt.Sprintf("every %dm", r.IntervalMinutes)
	}
	c.drafts[r.ID] = RoutineDraft{ID: r.ID, SessionID: c.state.SessionID, Revision: c.state.Revision, Edit: pluginrpc.DshbotRoutineEdit{Name: r.Name, Prompt: r.Prompt, Schedule: schedule, Timezone: r.Timezone, MaxRuns: r.MaxRuns, Enabled: r.Enabled}, MaxRunsText: strconv.Itoa(r.MaxRuns), Advanced: routineAdvancedFromRecord(r)}
	c.state.Review = nil
}
func (c *RoutineControls) Edit(s RoutineScope, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || c.state.Busy {
		return
	}
	d, ok := c.drafts[c.state.SelectedID]
	if !ok {
		return
	}
	switch field {
	case "name":
		d.Edit.Name = value
	case "prompt":
		d.Edit.Prompt = value
	case "schedule":
		d.Edit.Schedule = value
	case "timezone":
		d.Edit.Timezone = value
	case "maxRuns":
		d.MaxRunsText = value
	case "enabled":
		d.Edit.Enabled = value == "true"
	default:
		changed, err := editRoutineAdvanced(&d.Advanced, field, value, s.BotID, d.ID, c.state.Routines)
		if err != nil {
			c.state.Error = err.Error()
			c.state.Review = nil
			return
		}
		if !changed {
			return
		}
	}
	d.Version++
	c.drafts[d.ID] = d
	c.state.Review = nil
	c.state.Error = ""
}
func (c *RoutineControls) DiscardEdit(s RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(s) && !c.state.Busy {
		delete(c.drafts, c.state.SelectedID)
		c.state.Review = nil
		c.state.Error = ""
	}
}
func (c *RoutineControls) Dismiss(s RoutineScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(s) && !c.state.Busy {
		c.state.Review = nil
	}
}
func (c *RoutineControls) Prepare(s RoutineScope, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(s) || c.state.Busy || !c.state.Loaded {
		return
	}
	r, ok := c.row(c.state.SelectedID)
	if !ok {
		return
	}
	c.state.Review = nil
	cmd := pluginrpc.DshbotRoutineCommand{Action: action, ID: r.ID, BotID: s.BotID, Revision: c.state.Revision, EnabledBefore: r.Enabled}
	review := &RoutineReview{Command: cmd, SessionID: c.state.SessionID, Name: r.Name, Prompt: r.Prompt, Schedule: r.Schedule, Timezone: r.Timezone, WorkspaceID: c.state.WorkspaceID, Advanced: routineAdvancedFromRecord(r)}
	switch action {
	case "save":
		d, ok := c.drafts[r.ID]
		if !ok {
			return
		}
		if d.Revision != c.state.Revision || d.SessionID != c.state.SessionID {
			c.state.Error = "目录已变化。草稿保留；请核对当前记录后丢弃旧草稿再编辑 / Catalog changed; review current values before discarding the stale draft"
			return
		}
		n, err := strconv.Atoi(strings.TrimSpace(d.MaxRunsText))
		if err != nil {
			c.state.Error = "执行次数上限必须为整数 / Run limit must be an integer"
			return
		}
		d.Edit.MaxRuns = n
		if err := pluginrpc.ValidateDshbotRoutineEdit(d.Edit); err != nil {
			c.state.Error = err.Error()
			return
		}
		if d.Edit.Enabled && n > 0 && n <= r.RunCount {
			c.state.Error = "启用时次数上限必须大于已执行次数 / Limit must exceed current run count"
			return
		}
		advanced, err := prepareRoutineAdvanced(d.Advanced, s.BotID, r.ID, c.state.Routines)
		if err != nil {
			c.state.Error = err.Error()
			return
		}
		review.Command.Advanced = advanced
		review.Advanced = routineAdvancedReviewValue(d.Advanced, advanced)
		review.Command.Edit = d.Edit
		review.DraftVersion = d.Version
		review.Name = d.Edit.Name
		review.Prompt = d.Edit.Prompt
		review.Schedule = d.Edit.Schedule
		review.Timezone = d.Edit.Timezone
	case "run":
		if r.LastOutcome == "queued" || r.LastOutcome == "running" {
			c.state.Error = "上一轮仍在排队或运行 / A previous run is pending or running"
			return
		}
	case "toggle", "delete":
	default:
		return
	}
	c.reviewSequence++
	review.Sequence = c.reviewSequence
	c.state.Review = review
	c.state.Error = ""
	c.state.Notice = ""
}
func (c *RoutineControls) BeginAction(s RoutineScope, expected ...*RoutineReview) (RoutineTicket, pluginrpc.DshbotRoutineCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotRoutineCommand
	if !c.valid(s) || c.state.Busy || !c.state.Loaded || c.state.Review == nil {
		return RoutineTicket{}, empty, false
	}
	r := c.state.Review
	if len(expected) > 0 && (expected[0] == nil || expected[0].Sequence != r.Sequence) {
		return RoutineTicket{}, empty, false
	}
	if RoutineReviewNeedsWatchAcknowledgement(*r) && !r.WatchAcknowledged {
		return RoutineTicket{}, empty, false
	}
	cmd := r.Command
	cmd.Advanced = pluginrpc.CloneDshbotRoutineAdvanced(cmd.Advanced)
	if cmd.BotID != s.BotID || cmd.ID != c.state.SelectedID || cmd.Revision != c.state.Revision || r.SessionID != c.state.SessionID {
		return RoutineTicket{}, empty, false
	}
	row, ok := c.row(cmd.ID)
	if !ok || row.Enabled != cmd.EnabledBefore {
		return RoutineTicket{}, empty, false
	}
	if cmd.Action == "save" {
		d, ok := c.drafts[cmd.ID]
		if !ok || d.Version != r.DraftVersion || d.Revision != cmd.Revision {
			return RoutineTicket{}, empty, false
		}
	}
	c.clearRoutineTrigger()
	c.generation++
	c.state.Busy = true
	c.state.Error = ""
	return RoutineTicket{s, c.generation}, cmd, true
}
func (c *RoutineControls) FinishAction(t RoutineTicket, v *pluginrpc.DshbotRoutineView, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.matches(t) || c.state.Review == nil {
		return false
	}
	c.state.Busy = false
	review := *c.state.Review
	c.state.Review = nil
	if err == nil {
		err = pluginrpc.ValidateDshbotRoutineReceipt(review.Command, v)
	}
	if err == nil {
		b, _ := v.Bot(t.Scope.BotID)
		if b.SessionID != review.SessionID {
			err = fmt.Errorf("canonical Bot session changed; refresh before continuing")
		}
	}
	if err == nil {
		err = c.replace(v)
	}
	if err != nil {
		c.state.Loaded = false
		c.state.Error = err.Error() + "；结果可能已生效，先刷新核实再操作 / Outcome may have applied; refresh before another action"
		return true
	}
	if review.Command.Action == "save" || review.Command.Action == "delete" {
		delete(c.drafts, review.Command.ID)
	}
	if review.Command.Action == "delete" {
		c.state.SelectedID = ""
	}
	c.state.MissedKnown = false
	c.state.MissedIDs = nil
	c.state.Error = ""
	c.state.Notice = "Host 已返回目录 / Host returned catalog: " + review.Command.Action
	if review.Command.Action == "run" {
		c.state.Notice += "；执行结果请看状态与历史，不代表已完成 / Check status and history; this is not completion"
	}
	return true
}

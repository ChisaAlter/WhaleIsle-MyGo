package dshbot

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	"whaleisle/internal/desktop/pluginrpc"
)

// HostKey is an opaque fingerprint of the authenticated client, never a URL or
// persisted credential. Equal Bot IDs on different Hosts/accounts stay separate.
type TaskScope struct{ HostKey, BotID string }
type TaskTicket struct {
	Scope      TaskScope
	generation uint64
}
type TaskReview struct {
	Command   pluginrpc.DshbotTaskCommand
	SessionID string
}
type TaskControlSnapshot struct {
	Scope              TaskScope
	Loaded, Busy       bool
	Revision           int
	BotName, SessionID string
	Tasks              []pluginrpc.DshbotTask
	SelectedID         string
	Review             *TaskReview
	Error, Notice      string
}

type TaskControls struct {
	mu         sync.Mutex
	state      TaskControlSnapshot
	generation uint64
	retryIDs   map[TaskScope]map[string]string
}

func (c *TaskControls) Bind(scope TaskScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != scope {
		c.generation++
		c.state = TaskControlSnapshot{Scope: scope}
	}
}
func (c *TaskControls) Snapshot(scope TaskScope) TaskControlSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope != scope {
		return TaskControlSnapshot{Scope: scope}
	}
	out := c.state
	out.Tasks = append([]pluginrpc.DshbotTask(nil), out.Tasks...)
	if out.Review != nil {
		review := *out.Review
		out.Review = &review
	}
	return out
}
func (c *TaskControls) matches(t TaskTicket) bool {
	return c.state.Scope == t.Scope && c.generation == t.generation
}
func (c *TaskControls) valid(scope TaskScope) bool {
	return scope == c.state.Scope && scope.HostKey != "" && scope.BotID != ""
}
func (c *TaskControls) BeginLoad(scope TaskScope) (TaskTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(scope) || c.state.Busy {
		return TaskTicket{}, false
	}
	c.generation++
	c.state.Busy = true
	c.state.Error = ""
	return TaskTicket{scope, c.generation}, true
}
func (c *TaskControls) replaceView(view *pluginrpc.DshbotTaskView) error {
	bot, ok := view.Bot(c.state.Scope.BotID)
	if !ok {
		return fmt.Errorf("Bot is no longer available; refresh the roster")
	}
	if c.state.SessionID != "" && c.state.SessionID != bot.SessionID {
		c.state.Review = nil
		c.state.SelectedID = ""
	}
	c.state.BotName = bot.Name
	if c.state.BotName == "" {
		c.state.BotName = bot.ID
	}
	c.state.SessionID = bot.SessionID
	c.state.Revision = view.Revision
	c.state.Tasks = nil
	for _, task := range view.Value.Tasks {
		if task.ID != "" && (task.FromID == bot.ID || task.ToID == bot.ID) {
			c.state.Tasks = append(c.state.Tasks, task)
		}
	}
	sort.SliceStable(c.state.Tasks, func(i, j int) bool { return c.state.Tasks[i].UpdatedAt > c.state.Tasks[j].UpdatedAt })
	c.state.Loaded = true
	return nil
}
func (c *TaskControls) FinishLoad(t TaskTicket, view *pluginrpc.DshbotTaskView, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.matches(t) {
		return false
	}
	c.state.Busy = false
	if err == nil {
		err = c.replaceView(view)
	}
	if err != nil {
		c.state.Loaded = false
		c.state.Error = err.Error()
		return true
	}
	c.state.Error = ""
	c.state.Notice = "任务已从当前 Host 刷新 / Tasks refreshed"
	return true
}
func (c *TaskControls) Select(scope TaskScope, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(scope) || c.state.Busy {
		return
	}
	for _, task := range c.state.Tasks {
		if task.ID == id {
			c.state.SelectedID = id
			c.state.Review = nil
			c.state.Error = ""
			return
		}
	}
}
func (c *TaskControls) task(id string) (pluginrpc.DshbotTask, bool) {
	for _, task := range c.state.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return pluginrpc.DshbotTask{}, false
}
func (c *TaskControls) Prepare(scope TaskScope, action, taskID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(scope) || c.state.Busy || !c.state.Loaded {
		return
	}
	command := pluginrpc.DshbotTaskCommand{Action: action, TaskID: taskID, BotID: scope.BotID, Revision: c.state.Revision}
	if action != "stop" {
		task, ok := c.task(taskID)
		if !ok || taskID != c.state.SelectedID || !pluginrpc.DshbotTaskActionAllowed(action, task.Status) {
			return
		}
	} else {
		command.TaskID = ""
	}
	if action == "retry" {
		if c.retryIDs == nil {
			c.retryIDs = make(map[TaskScope]map[string]string)
		}
		ids := c.retryIDs[scope]
		if ids == nil {
			ids = make(map[string]string)
			c.retryIDs[scope] = ids
		}
		if ids[taskID] == "" {
			var bytes [16]byte
			if _, err := rand.Read(bytes[:]); err != nil {
				c.state.Error = "Could not create retry request ID: " + err.Error()
				return
			}
			ids[taskID] = "mygo-task-" + hex.EncodeToString(bytes[:])
		}
		command.RequestID = ids[taskID]
	}
	c.state.Review = &TaskReview{Command: command, SessionID: c.state.SessionID}
	c.state.Error = ""
	c.state.Notice = ""
}
func (c *TaskControls) Dismiss(scope TaskScope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(scope) || c.state.Busy {
		return
	}
	// Keep an ambiguous retry's operation ID even after dismissing the review.
	c.state.Review = nil
	c.state.Error = ""
}
func (c *TaskControls) BeginAction(scope TaskScope) (TaskTicket, pluginrpc.DshbotTaskCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var empty pluginrpc.DshbotTaskCommand
	if !c.valid(scope) || c.state.Busy || !c.state.Loaded || c.state.Review == nil {
		return TaskTicket{}, empty, false
	}
	review := c.state.Review
	command := review.Command
	if command.Revision != c.state.Revision || review.SessionID != c.state.SessionID {
		c.state.Error = "目录已变化。关闭确认框并重新核对操作 / Catalog changed; review the action again"
		return TaskTicket{}, empty, false
	}
	if command.Action != "stop" {
		task, ok := c.task(command.TaskID)
		if !ok || !pluginrpc.DshbotTaskActionAllowed(command.Action, task.Status) {
			c.state.Error = "Task status changed; refresh and review again"
			return TaskTicket{}, empty, false
		}
	}
	c.generation++
	c.state.Busy = true
	c.state.Error = ""
	return TaskTicket{scope, c.generation}, command, true
}
func (c *TaskControls) FinishAction(t TaskTicket, result pluginrpc.DshbotTaskResult, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.matches(t) || c.state.Review == nil {
		return false
	}
	c.state.Busy = false
	review := *c.state.Review
	if err == nil {
		bot, ok := result.View.Bot(c.state.Scope.BotID)
		if !ok || bot.SessionID != review.SessionID {
			err = fmt.Errorf("Bot identity changed; refresh before continuing")
		}
	}
	if err == nil {
		err = c.replaceView(result.View)
	}
	if err != nil {
		c.state.Error = err.Error()
		return true
	}
	if review.Command.Action == "retry" {
		delete(c.retryIDs[c.state.Scope], review.Command.TaskID)
		c.state.SelectedID = result.TaskID
	}
	c.state.Review = nil
	c.state.Error = ""
	c.state.Notice = "Host 已确认 / Host accepted: " + review.Command.Action
	if strings.TrimSpace(result.WakeError) != "" {
		c.state.Notice += "；任务状态已保存，但唤醒失败 / State saved; wake failed: " + result.WakeError
	} else if (review.Command.Action == "resume" || review.Command.Action == "retry") && !result.WakeAccepted {
		c.state.Notice += "；Host 未确认唤醒，请刷新检查 / Wake not confirmed; refresh to inspect"
	}
	return true
}

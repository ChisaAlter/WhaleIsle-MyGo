package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// CreatorTask is an explicit new-session admission, independent of persisted
// defaults and the active-session preset picker. It never submits a prompt.
type CreatorTaskState struct {
	Ticket                                            uint64
	Review, Busy, Attempted, Created, Applied, Opened bool
	SessionID, Host, Cwd, Phase, Error                string
}
type creatorTaskOwner struct {
	client              *dsh.Client
	rpc                 *dsh.Client
	endpoint            dsh.Endpoint
	epoch, follow       uint64
	selected, workspace string
}
type CreatorTask struct {
	mu        sync.Mutex
	source    *Store
	visible   bool
	isVisible func() bool
	owner     creatorTaskOwner
	state     CreatorTaskState
}

func NewCreatorTask(source *Store, isVisible func() bool) *CreatorTask {
	return &CreatorTask{source: source, isVisible: isVisible}
}
func (c *CreatorTask) Snapshot() CreatorTaskState {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Review && !c.currentLocked(c.state.Ticket, true) {
		c.state.Review = false
		c.state.Ticket++
		c.state.Error = "Creator 确认已失效；请在当前 Host 和页面重新准备 / Creator confirmation expired"
	}
	return c.state
}

// A departure invalidates every callback from that rendered view. Receipts stay
// visible on return, including a mutation that finished after the departure.
func (c *CreatorTask) ObserveVisible(visible bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.visible && !visible {
		c.state.Ticket++
		c.state.Review = false
	}
	c.visible = visible
}
func (c *CreatorTask) viewCurrentLocked(ticket uint64) bool {
	return c.visible && c.state.Ticket == ticket && (c.isVisible == nil || c.isVisible())
}
func (c *CreatorTask) ownerCurrentLocked(original bool) bool {
	s := c.source
	return s.client == c.owner.client && s.client != nil && s.client.Endpoint == c.owner.endpoint && s.composerHostGeneration == c.owner.epoch && (!original || (s.selectedID == c.owner.selected && s.followGeneration == c.owner.follow && s.cwd == c.owner.workspace))
}
func (c *CreatorTask) currentLocked(ticket uint64, original bool) bool {
	if c.source == nil || !c.viewCurrentLocked(ticket) {
		return false
	}
	c.source.mu.Lock()
	defer c.source.mu.Unlock()
	return c.ownerCurrentLocked(original)
}
func creatorHostLabel(origin string) string {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return "Host"
	}
	return u.Scheme + "://" + u.Host + u.EscapedPath()
}
func (c *CreatorTask) Prepare() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.source == nil || !c.visible || (c.isVisible != nil && !c.isVisible()) || c.state.Busy || (c.state.Attempted && !c.state.Opened) {
		return
	}
	s := c.source
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		c.state.Error = "请先连接 Host / Connect a Host first"
		return
	}
	cwd := s.cwd
	if strings.TrimSpace(cwd) == "" && s.workspaceReady {
		cwd = s.scratchCwd
	}
	if strings.TrimSpace(cwd) == "" {
		c.state.Error = "请选择工作目录，或等待 Host 提供无目录工作区 / Choose a workspace first"
		return
	}
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		c.state.Error = "无法生成新任务身份 / Cannot allocate a task identity"
		return
	}
	uuid[6] = (uuid[6] & 15) | 64
	uuid[8] = (uuid[8] & 63) | 128
	id := fmt.Sprintf("session-%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
	c.owner = creatorTaskOwner{client: s.client, rpc: &dsh.Client{Endpoint: s.client.Endpoint, HTTP: s.client.HTTP}, endpoint: s.client.Endpoint, epoch: s.composerHostGeneration, follow: s.followGeneration, selected: s.selectedID, workspace: s.cwd}
	c.state = CreatorTaskState{Ticket: c.state.Ticket + 1, Review: true, SessionID: id, Host: creatorHostLabel(s.client.Endpoint.Origin), Cwd: cwd}
}
func (c *CreatorTask) Cancel(ticket uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Ticket == ticket && c.state.Review && !c.state.Busy {
		c.state.Review = false
		c.state.Ticket++
	}
}
func creatorAvailable(roster api.AgentPresetRoster) bool {
	for _, p := range roster.Presets {
		if p.Id == "cordis" {
			return (p.Hidden == nil || !*p.Hidden) && (p.Broken == nil || strings.TrimSpace(*p.Broken) == "")
		}
	}
	return false
}
func (c *CreatorTask) complete(id, phase, message string) {
	c.mu.Lock()
	if c.state.SessionID == id {
		c.state.Busy = false
		c.state.Phase = phase
		c.state.Error = message
	}
	c.mu.Unlock()
	c.source.fire()
}

// Create consumes one review. The caller supplies a bounded context. A fixed
// request.sessionId is part of the original create contract, so even a lost
// receipt has a durable identity. Neither create nor select is ever retried.
func (c *CreatorTask) Create(ctx context.Context, ticket uint64) bool {
	c.mu.Lock()
	if !c.state.Review || c.state.Busy || c.state.Attempted || ctx.Err() != nil || !c.currentLocked(ticket, true) {
		c.mu.Unlock()
		return false
	}
	c.state.Review = false
	c.state.Busy = true
	c.state.Error = ""
	c.state.Phase = "checking"
	id, cwd, client := c.state.SessionID, c.state.Cwd, c.owner.rpc
	c.mu.Unlock()
	c.source.fire()
	var roster api.AgentPresetRoster
	if err := client.Call(ctx, "agentPresets/list", map[string]any{}, &roster); err != nil {
		c.complete(id, "checking", "无法确认 cordis 是否可用；尚未创建任务 / Could not verify cordis; no task was created")
		return false
	}
	if !creatorAvailable(roster) {
		c.complete(id, "checking", "当前 Host 的 cordis 缺失、隐藏或损坏；尚未创建任务 / cordis is unavailable; no task was created")
		return false
	}
	c.mu.Lock()
	if ctx.Err() != nil || !c.currentLocked(ticket, true) {
		c.mu.Unlock()
		c.complete(id, "checking", "页面或 Host 已改变，未创建任务 / Navigation changed; no task was created")
		return false
	}
	c.state.Attempted = true
	c.state.Phase = "creating"
	c.mu.Unlock()
	c.source.fire()
	var out api.SessionCreateValue
	err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &cwd, SessionId: &id}}, &out)
	if err != nil || out.SessionId != id {
		c.complete(id, "create-unknown", "创建结果未确认；保留目标 ID，请定位已有任务，勿重复创建 / Creation outcome unknown; locate the retained ID before creating again")
		return false
	}
	c.mu.Lock()
	c.state.Created = true
	c.state.Phase = "created"
	if ctx.Err() != nil || !c.currentLocked(ticket, true) {
		c.mu.Unlock()
		c.complete(id, "created", "任务已创建，页面或 Host 已改变；未应用 cordis / Task created; navigation changed before cordis was applied")
		return false
	}
	c.state.Phase = "applying"
	c.mu.Unlock()
	var selected api.SelectResult
	err = client.Call(ctx, "agentPresets/select", map[string]any{"agentId": id, "agentPreset": "cordis"}, &selected)
	if err != nil || string(selected) != "cordis" {
		c.complete(id, "apply-unknown", "任务已创建，cordis 应用结果未确认；可打开该任务检查，不会自动重试 / Task created; cordis outcome unconfirmed; open the existing task to inspect")
		return false
	}
	c.mu.Lock()
	c.state.Applied = true
	current := ctx.Err() == nil && c.currentLocked(ticket, true)
	c.mu.Unlock()
	if !current {
		c.complete(id, "ready", "已在该任务应用 cordis；页面已改变，未切换当前会话 / cordis applied; newer navigation preserved")
		return false
	}
	c.complete(id, "ready", "")
	return true
}

// Locate only reads the exact attempted identity. It never adopts an unrelated
// blank session, reapplies a preset, or treats an absent result as safe to retry.
func (c *CreatorTask) Locate(ctx context.Context, ticket uint64) {
	c.mu.Lock()
	if c.state.Busy || !c.state.Attempted || c.state.Created || ctx.Err() != nil || !c.currentLocked(ticket, false) {
		c.mu.Unlock()
		return
	}
	id, cwd, client := c.state.SessionID, c.state.Cwd, c.owner.rpc
	c.state.Busy = true
	c.mu.Unlock()
	c.source.fire()
	var list api.SessionListValue
	if err := client.Call(ctx, "session/list", map[string]any{"_request": api.SessionListRequest{}}, &list); err != nil {
		c.complete(id, "create-unknown", "读取任务列表失败；目标 ID 保留 / Could not locate task; ID retained")
		return
	}
	found := false
	for _, row := range list.Items {
		if row.SessionId == id && row.Cwd != nil && filepathClean(*row.Cwd) == filepathClean(cwd) {
			found = true
			break
		}
	}
	c.mu.Lock()
	if !c.currentLocked(ticket, false) {
		c.mu.Unlock()
		c.complete(id, "create-unknown", "Host 或页面已改变；定位结果未采纳 / Host or page changed; readback not adopted")
		return
	}
	c.state.Created = found
	c.mu.Unlock()
	if found {
		c.complete(id, "located", "已定位此目标；cordis 状态未确认，请打开检查 / Existing task located; inspect its preset in the composer")
	} else {
		c.complete(id, "create-unknown", "列表尚未确认此目标；不会据此重建 / This ID is not confirmed in the list; no replacement was created")
	}
}

// Open atomically commits only the retained target. Automatic navigation must
// still belong to the original view and selection; explicit recovery may open
// the same confirmed target after returning to this pane on the same Host.
func (c *CreatorTask) Open(ticket uint64, automatic bool) bool {
	c.mu.Lock()
	if c.state.Busy || !c.state.Created || (automatic && !c.state.Applied) || !c.viewCurrentLocked(ticket) {
		c.mu.Unlock()
		return false
	}
	s := c.source
	s.mu.Lock()
	if !c.ownerCurrentLocked(automatic) {
		s.mu.Unlock()
		c.mu.Unlock()
		return false
	}
	id, cwd := c.state.SessionID, c.state.Cwd
	found := false
	for _, item := range s.items {
		if item.SessionID == id {
			found = true
			break
		}
	}
	if !found {
		s.items = append(s.items, Item{SessionID: id, Cwd: cwd, Blank: true})
	}
	s.selectSessionLocked(id)
	s.selectedAgentPresetLabel = ""
	if c.state.Applied {
		s.selectedAgentPresetLabel = labelForAgentPresetID("cordis", s.agentPresetOptions)
	}
	s.lastAction = "Creator task: " + id
	c.state.Opened = true
	// Bind observers under the same lock as the exact target selection. Never
	// acquire the current selection/client again after publishing this change.
	client, mux, generation := s.client, s.mux, s.followGeneration
	var followCtx, controlCtx context.Context
	if mux != nil {
		followCtx, s.followCancel = context.WithCancel(context.Background())
		if s.controlCancel != nil {
			s.controlCancel()
		}
		controlCtx, s.controlCancel = context.WithCancel(context.Background())
		s.controlNote = ""
	}
	s.mu.Unlock()
	c.mu.Unlock()
	s.fire()
	if mux != nil {
		go s.runFollow(followCtx, mux, id, generation)
		go s.runControlWatch(controlCtx, client, mux, id)
	}
	return true
}

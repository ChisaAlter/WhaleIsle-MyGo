package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoomRuntimeSnapshot struct {
	Scope                string
	Generation, Sequence uint64
	Loaded, Busy         bool
	Data                 pluginrpc.DshbotRoomRuntime
	Error, Notice        string
}
type RoomRuntimeTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type RoomRuntimeReader struct {
	mu    sync.Mutex
	state RoomRuntimeSnapshot
}

func (c *RoomRuntimeReader) Bind(scope string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Scope == scope {
		return
	}
	generation := c.state.Generation + 1
	c.state = RoomRuntimeSnapshot{Scope: scope, Generation: generation}
}
func (c *RoomRuntimeReader) Suspend() { c.Bind("") }
func (c *RoomRuntimeReader) Snapshot() RoomRuntimeSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := c.state
	v.Data = pluginrpc.CloneDshbotRoomRuntime(v.Data)
	return v
}
func (c *RoomRuntimeReader) ViewCurrent(scope string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return scope != "" && c.state.Scope == scope && c.state.Generation == g
}
func (c *RoomRuntimeReader) BeginRead(scope string, g uint64) (RoomRuntimeTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := &c.state
	if scope == "" || s.Scope != scope || s.Generation != g || s.Busy {
		return RoomRuntimeTicket{}, false
	}
	s.Sequence++
	s.Busy = true
	s.Loaded = false
	s.Error = ""
	s.Notice = ""
	return RoomRuntimeTicket{scope, g, s.Sequence}, true
}
func (c *RoomRuntimeReader) current(t RoomRuntimeTicket) bool {
	s := c.state
	return t.Scope != "" && s.Scope == t.Scope && s.Generation == t.Generation && s.Sequence == t.Sequence && s.Busy
}
func (c *RoomRuntimeReader) Current(t RoomRuntimeTicket) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current(t)
}
func (c *RoomRuntimeReader) FinishRead(t RoomRuntimeTicket, v pluginrpc.DshbotRoomRuntime, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	s := &c.state
	s.Busy = false
	s.Loaded = err == nil
	if err != nil {
		s.Error = err.Error()
		s.Notice = "运行态不可用；旧快照不代表当前状态 / Runtime unavailable; any old snapshot is stale"
	} else {
		s.Data = pluginrpc.CloneDshbotRoomRuntime(v)
		s.Error = ""
		s.Notice = "只读快照；不会自动 follow 或启动成员 / Read-only snapshot; no automatic member activation"
	}
	return true
}
func (c *RoomRuntimeReader) BeginOpen(scope string, g uint64, sessionID string) (RoomRuntimeTicket, pluginrpc.DshbotRoomRuntime, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := &c.state
	if scope == "" || s.Scope != scope || s.Generation != g || !s.Loaded || s.Busy {
		return RoomRuntimeTicket{}, pluginrpc.DshbotRoomRuntime{}, false
	}
	if _, err := pluginrpc.DshbotRoomRuntimeTarget(s.Data, sessionID); err != nil {
		s.Error = err.Error()
		return RoomRuntimeTicket{}, pluginrpc.DshbotRoomRuntime{}, false
	}
	s.Sequence++
	s.Busy = true
	s.Error = ""
	return RoomRuntimeTicket{scope, g, s.Sequence}, pluginrpc.CloneDshbotRoomRuntime(s.Data), true
}
func (c *RoomRuntimeReader) FinishOpen(t RoomRuntimeTicket, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.current(t) {
		return false
	}
	c.state.Busy = false
	if err != nil {
		c.state.Error = err.Error()
		c.state.Loaded = false
		c.state.Notice = "打开失败；未创建或执行成员会话，请重新读取状态 / Open failed; refresh before another explicit open"
	}
	return true
}

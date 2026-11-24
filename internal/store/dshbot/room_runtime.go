package dshbot

import (
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type RoomRuntimeSnapshot struct {
	Scope                string
	Generation, Sequence uint64
	Loaded, Busy         bool
	Data                 pluginrpc.DshbotRoomRuntime
	Error                *state.Fault
	Notice               string
}
type RoomRuntimeTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type RoomRuntimeReader struct {
	mu    sync.Mutex
	state RoomRuntimeSnapshot
	pub   state.Published[RoomRuntimeSnapshot]
	dirty bool
}

func (c *RoomRuntimeReader) Bind(scope string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirty = true
	if c.state.Scope == scope {
		return
	}
	generation := c.state.Generation + 1
	c.state = RoomRuntimeSnapshot{Scope: scope, Generation: generation}
}
func (c *RoomRuntimeReader) Suspend() { c.Bind("") }
func (c *RoomRuntimeReader) Snapshot() *state.Version[RoomRuntimeSnapshot] {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v := c.pub.Load(); v != nil && !c.dirty {
		return v
	}
	out := c.state
	out.Data = pluginrpc.CloneDshbotRoomRuntime(out.Data)
	c.pub.Publish(out)
	c.dirty = false
	return c.pub.Load()
}
func (c *RoomRuntimeReader) ViewCurrent(scope string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return scope != "" && c.state.Scope == scope && c.state.Generation == g
}
func (c *RoomRuntimeReader) BeginRead(scope string, g uint64) (RoomRuntimeTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirty = true
	s := &c.state
	if scope == "" || s.Scope != scope || s.Generation != g || s.Busy {
		return RoomRuntimeTicket{}, false
	}
	s.Sequence++
	s.Busy = true
	s.Loaded = false
	s.Error = nil
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
	c.dirty = true
	if !c.current(t) {
		return false
	}
	s := &c.state
	s.Busy = false
	s.Loaded = err == nil
	if err != nil {
		s.Error = state.FaultOf(err)
		s.Notice = storelocale.T(storelocale.KeyStoreDshbotRuntimeUnavailableAnyOldSnapshotIs)
	} else {
		s.Data = pluginrpc.CloneDshbotRoomRuntime(v)
		s.Error = nil
		s.Notice = storelocale.T(storelocale.KeyStoreDshbotReadOnlySnapshotNoAutomaticMemberA)
	}
	return true
}
func (c *RoomRuntimeReader) BeginOpen(scope string, g uint64, sessionID string) (RoomRuntimeTicket, pluginrpc.DshbotRoomRuntime, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirty = true
	s := &c.state
	if scope == "" || s.Scope != scope || s.Generation != g || !s.Loaded || s.Busy {
		return RoomRuntimeTicket{}, pluginrpc.DshbotRoomRuntime{}, false
	}
	if _, err := pluginrpc.DshbotRoomRuntimeTarget(s.Data, sessionID); err != nil {
		s.Error = state.FaultOf(err)
		return RoomRuntimeTicket{}, pluginrpc.DshbotRoomRuntime{}, false
	}
	s.Sequence++
	s.Busy = true
	s.Error = nil
	return RoomRuntimeTicket{scope, g, s.Sequence}, pluginrpc.CloneDshbotRoomRuntime(s.Data), true
}
func (c *RoomRuntimeReader) FinishOpen(t RoomRuntimeTicket, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirty = true
	if !c.current(t) {
		return false
	}
	c.state.Busy = false
	if err != nil {
		c.state.Error = state.FaultOf(err)
		c.state.Loaded = false
		c.state.Notice = storelocale.T(storelocale.KeyStoreDshbotOpenFailedRefreshBeforeAnotherExpli)
	}
	return true
}

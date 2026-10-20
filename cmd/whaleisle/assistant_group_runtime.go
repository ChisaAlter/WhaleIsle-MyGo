package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type assistantGroupLiveState struct {
	mu                sync.Mutex
	scope             string
	busy, loaded      bool
	at                time.Time
	data              pluginrpc.DshbotRoomRuntime
	err               string
	rendered, visible bool
	generation        uint64
	cancel            context.CancelFunc
	timer             *time.Timer
	contacts          assistantContactRoomLiveState
}

// Frame ownership follows the actual painted page, including embedded views.
func (s *mainState) beginAssistantGroupRuntimeFrame() {
	live := &s.assistantGroupLive
	live.mu.Lock()
	live.rendered = false
	live.mu.Unlock()
	live.contacts.beginFrame()
}

func (s *mainState) finishAssistantGroupRuntimeFrame() {
	live := &s.assistantGroupLive
	live.contacts.finishFrame()
	live.mu.Lock()
	defer live.mu.Unlock()
	if live.rendered || !live.visible {
		return
	}
	live.visible = false
	live.generation++
	live.scope = ""
	live.busy, live.loaded = false, false
	if live.cancel != nil {
		live.cancel()
		live.cancel = nil
	}
	if live.timer != nil {
		live.timer.Stop()
		live.timer = nil
	}
}

// Read-only runtime refresh is bounded to the visibly selected canonical room.
// A scope change makes both the pending result and its repaint timer obsolete.
func (s *mainState) refreshAssistantGroupRuntime(ep harness.Endpoint, roomID, sessionID string, members []session.GroupMember) []session.GroupMember {
	s.mu.Lock()
	sessions, win := s.sessions, s.win
	s.mu.Unlock()
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if sessions == nil || client == nil || win == nil || client.Endpoint.Origin != ep.Origin || client.Endpoint.Cookie != ep.Cookie {
		return members
	}
	snap := sessions.Snapshot()
	if snap.SelectedID != sessionID {
		return members
	}
	running := snap.PromptBusy
	for _, item := range snap.Items {
		if item.SessionID == sessionID && item.Running {
			running = true
		}
	}
	scope := fmt.Sprintf("%p:%s:%s:%s:%d", client, ep.Origin, ep.Cookie, sessionID, snap.ComposerHostGeneration)
	live := &s.assistantGroupLive
	live.mu.Lock()
	live.rendered, live.visible = true, true
	if live.scope != scope {
		if live.cancel != nil {
			live.cancel()
			live.cancel = nil
		}
		if live.timer != nil {
			live.timer.Stop()
			live.timer = nil
		}
		live.generation++
		live.scope = scope
		live.busy = false
		live.loaded = false
		live.at = time.Time{}
		live.data = pluginrpc.DshbotRoomRuntime{}
		live.err = ""
	}
	start := !live.busy && (!live.loaded || running && time.Since(live.at) >= 1500*time.Millisecond)
	var readContext context.Context
	var cancelRead context.CancelFunc
	if start {
		live.busy = true
		readContext, cancelRead = context.WithTimeout(context.Background(), 15*time.Second)
		live.cancel = cancelRead
	}
	generation := live.generation
	data, loadErr := live.data, live.err
	live.mu.Unlock()
	for i := range members {
		members[i].RuntimeError = loadErr
		for _, member := range data.Members {
			if member.BotID == members[i].ID {
				members[i].RuntimeStatus = member.Status
				if pluginrpc.DshbotRoomRuntimeAttention(member) {
					members[i].RuntimeError = i18n.T(i18n.KeyCmdNeedsAttention)
				}
				break
			}
		}
	}
	if !start {
		return members
	}
	rpc := s.pluginRPCClient(ep)
	current := func() bool {
		live.mu.Lock()
		visible := live.visible && live.scope == scope && live.generation == generation
		live.mu.Unlock()
		if !visible {
			return false
		}
		s.dshMu.Lock()
		same := s.dshClient == client
		s.dshMu.Unlock()
		if !same {
			return false
		}
		s.mu.Lock()
		valid := s.sessions == sessions && s.bootStatus.State == harness.Ready && s.bootStatus.Endpoint == ep && s.cfgStore != nil && s.cfgStore.Get().DshbotEnabled
		s.mu.Unlock()
		selected := sessions.Snapshot()
		return valid && selected.SelectedID == sessionID && selected.ComposerHostGeneration == snap.ComposerHostGeneration
	}
	go func() {
		defer cancelRead()
		result, err := rpc.DshbotRoomRuntime(readContext, roomID)
		if !current() {
			live.mu.Lock()
			if live.scope == scope && live.generation == generation {
				live.busy = false
				live.loaded = false
			}
			live.mu.Unlock()
			return
		}
		live.mu.Lock()
		if live.scope != scope || live.generation != generation || !live.visible {
			live.mu.Unlock()
			return
		}
		live.busy = false
		live.cancel = nil
		live.loaded = true
		live.at = time.Now()
		live.err = ""
		if err != nil {
			live.err = err.Error()
		} else {
			live.data = result
		}
		if live.timer != nil {
			live.timer.Stop()
		}
		if running {
			live.timer = time.AfterFunc(1500*time.Millisecond, func() {
				if current() {
					win.Update(func() {})
				}
			})
		}
		live.mu.Unlock()
		win.Update(func() {})
	}()
	return members
}

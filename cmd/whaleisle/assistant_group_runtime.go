package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

type assistantGroupLiveState struct {
	mu           sync.Mutex
	scope        string
	busy, loaded bool
	at           time.Time
	data         pluginrpc.DshbotRoomRuntime
	err          string
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
	if live.scope != scope {
		live.scope = scope
		live.busy = false
		live.loaded = false
		live.at = time.Time{}
		live.data = pluginrpc.DshbotRoomRuntime{}
		live.err = ""
	}
	start := !live.busy && (!live.loaded || (running && time.Since(live.at) >= 1500*time.Millisecond))
	if start {
		live.busy = true
	}
	data, loadErr := live.data, live.err
	live.mu.Unlock()
	for i := range members {
		members[i].RuntimeError = loadErr
		for _, member := range data.Members {
			if member.BotID == members[i].ID {
				members[i].RuntimeStatus = member.Status
				if pluginrpc.DshbotRoomRuntimeAttention(member) {
					members[i].RuntimeError = "需要处理 / Needs attention"
				}
				break
			}
		}
	}
	if !start {
		return members
	}
	current := func() bool {
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
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		result, err := s.pluginRPCClient(ep).DshbotRoomRuntime(ctx, roomID)
		if !current() {
			live.mu.Lock()
			if live.scope == scope {
				live.busy = false
				live.loaded = false
			}
			live.mu.Unlock()
			return
		}
		live.mu.Lock()
		if live.scope != scope {
			live.mu.Unlock()
			return
		}
		live.busy = false
		live.loaded = true
		live.at = time.Now()
		live.data = result
		live.err = ""
		if err != nil {
			live.err = err.Error()
		}
		live.mu.Unlock()
		win.Update(func() {})
		if running {
			time.AfterFunc(1500*time.Millisecond, func() {
				if current() {
					win.Update(func() {})
				}
			})
		}
	}()
	return members
}

package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
)

type assistantContactRoomRuntime struct {
	data pluginrpc.DshbotRoomRuntime
	err  string
}

type assistantContactRoomLiveState struct {
	mu                              sync.Mutex
	scope                           string
	generation                      uint64
	rendered, visible, busy, loaded bool
	at                              time.Time
	rows                            map[string]assistantContactRoomRuntime
	cancel                          context.CancelFunc
	timer                           *time.Timer
}

func (live *assistantContactRoomLiveState) beginFrame() {
	live.mu.Lock()
	live.rendered = false
	live.mu.Unlock()
}

func (live *assistantContactRoomLiveState) stopLocked() {
	live.generation++
	live.scope = ""
	live.visible, live.busy, live.loaded = false, false, false
	live.rows = nil
	if live.cancel != nil {
		live.cancel()
		live.cancel = nil
	}
	if live.timer != nil {
		live.timer.Stop()
		live.timer = nil
	}
}

func (live *assistantContactRoomLiveState) finishFrame() {
	live.mu.Lock()
	defer live.mu.Unlock()
	if !live.rendered && live.visible {
		live.stopLocked()
	}
}

// Original BotPage polls every known room while mounted, including its task
// and routine views. The native assistant mounts those blocks together.
func (s *mainState) assistantContactRoomRuntimes(ep harness.Endpoint, catalog pluginrpc.DshbotContactCatalog) map[string]assistantContactRoomRuntime {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	assistant, sessions, win := s.assistant, s.sessions, s.win
	s.mu.Unlock()
	if client == nil || assistant == nil || sessions == nil || win == nil {
		return nil
	}
	selected := sessions.Snapshot()
	var identities []string
	targets := map[string]string{}
	interval := 5 * time.Second
	for _, row := range catalog.Items {
		if row.Kind != "room" || row.ID == "" || row.SessionID == "" {
			continue
		}
		targets[row.ID] = row.SessionID
		identities = append(identities, row.ID+":"+row.SessionID)
		if row.SessionID == selected.SelectedID {
			interval = 1500 * time.Millisecond
		}
	}
	if len(targets) == 0 {
		return nil
	}
	sort.Strings(identities)
	scope := fmt.Sprintf("%p:%s:%s:%s:%s", client, ep.Origin, ep.Cookie, selected.SelectedID, strings.Join(identities, "\x00"))
	live := &s.assistantGroupLive.contacts
	live.mu.Lock()
	if live.scope != scope {
		live.stopLocked()
		live.scope = scope
		live.rows = map[string]assistantContactRoomRuntime{}
	}
	live.rendered, live.visible = true, true
	start := !live.busy && (!live.loaded || time.Since(live.at) >= interval)
	var ctx context.Context
	var cancel context.CancelFunc
	if start {
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		live.cancel = cancel
		live.busy = true
	}
	generation := live.generation
	out := make(map[string]assistantContactRoomRuntime, len(live.rows))
	for id, row := range live.rows {
		out[id] = row
	}
	live.mu.Unlock()
	if !start {
		return out
	}
	current := func() bool {
		live.mu.Lock()
		visible := live.visible && live.scope == scope && live.generation == generation
		live.mu.Unlock()
		if !visible || s.assistantRoomClient(ep) != client {
			return false
		}
		s.mu.Lock()
		same := s.assistant == assistant && s.sessions == sessions
		s.mu.Unlock()
		now := sessions.Snapshot()
		return same && now.SelectedID == selected.SelectedID
	}
	rpc := s.pluginRPCClient(ep)
	go func() {
		defer cancel()
		type result struct {
			id   string
			data pluginrpc.DshbotRoomRuntime
			err  error
		}
		results := make(chan result, len(targets))
		// Match the original independent per-room reads; no room waits behind
		// an unrelated room's timeout. A single owned context cancels the batch.
		for id := range targets {
			go func(id string) {
				data, err := rpc.DshbotRoomRuntime(ctx, id)
				results <- result{id: id, data: data, err: err}
			}(id)
		}
		fresh := map[string]result{}
		for range targets {
			row := <-results
			fresh[row.id] = row
		}
		if !current() {
			return
		}
		live.mu.Lock()
		if !live.visible || live.scope != scope || live.generation != generation {
			live.mu.Unlock()
			return
		}
		for id, result := range fresh {
			row := live.rows[id]
			row.err = ""
			if result.err != nil {
				row.err = result.err.Error()
			} else if result.data.Room.ID != id || result.data.Room.SessionID != targets[id] {
				row.err = "Room identity changed; refresh the contact catalog"
			} else {
				row.data = result.data
			}
			live.rows[id] = row
		}
		live.busy, live.loaded = false, true
		live.cancel = nil
		live.at = time.Now()
		if live.timer != nil {
			live.timer.Stop()
		}
		live.timer = time.AfterFunc(interval, func() {
			if current() {
				win.Update(func() {})
			}
		})
		live.mu.Unlock()
		win.Update(func() {})
	}()
	return out
}

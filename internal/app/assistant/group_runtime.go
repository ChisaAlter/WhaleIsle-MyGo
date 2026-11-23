package assistantctl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type contactRoomRuntime struct {
	data pluginrpc.DshbotRoomRuntime
	err  string
}

type contactRoomLiveState struct {
	mu                              sync.Mutex
	scope                           string
	generation                      uint64
	rendered, visible, busy, loaded bool
	at                              time.Time
	rows                            map[string]contactRoomRuntime
	cancel                          context.CancelFunc
	timer                           *time.Timer
}

func (live *contactRoomLiveState) beginFrame() {
	live.mu.Lock()
	live.rendered = false
	live.mu.Unlock()
}

func (live *contactRoomLiveState) stopLocked() {
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

func (live *contactRoomLiveState) stop() {
	live.mu.Lock()
	live.stopLocked()
	live.mu.Unlock()
}

func (live *contactRoomLiveState) finishFrame() {
	live.mu.Lock()
	defer live.mu.Unlock()
	if !live.rendered && live.visible {
		live.stopLocked()
	}
}

type groupLiveState struct {
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
	contacts          contactRoomLiveState
}

func (live *groupLiveState) stopLocked() {
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

// BeginGroupRuntimeFrame clears per-frame paint flags before rendering.
func (s *Service) BeginGroupRuntimeFrame() {
	if s == nil {
		return
	}
	live := &s.assistantGroupLive
	live.mu.Lock()
	live.rendered = false
	live.mu.Unlock()
	live.contacts.beginFrame()
}

// FinishGroupRuntimeFrame cancels any room/contacts runtime readers that did
// not actually paint during the frame.
func (s *Service) FinishGroupRuntimeFrame() {
	if s == nil {
		return
	}
	live := &s.assistantGroupLive
	live.contacts.finishFrame()
	live.mu.Lock()
	defer live.mu.Unlock()
	if live.rendered || !live.visible {
		return
	}
	live.stopLocked()
}

// StopGroupRuntime stops both room runtime and contact-list room pollers.
func (s *Service) StopGroupRuntime() {
	if s == nil {
		return
	}
	live := &s.assistantGroupLive
	live.mu.Lock()
	live.stopLocked()
	live.mu.Unlock()
	live.contacts.stop()
}

// GroupMembers matches the selected canonical session against the current Host's
// Dshbot catalog and enriches group rooms with live member runtime status.
func (s *Service) GroupMembers(sessionID string) (bool, []session.GroupMember) {
	if s == nil {
		return false, nil
	}
	host := s.safeHost()
	roster := s.DshbotRoster()
	ep, bootState, _ := host.BootEndpoint()
	ready := bootState == harness.Ready
	cfg := host.Config()
	if roster == nil || cfg == nil || !cfg.Get().DshbotEnabled || !ready || sessionID == "" {
		return false, nil
	}
	snap := roster.Snapshot().Value
	if snap.Origin != ep.Origin {
		return false, nil
	}
	for _, room := range snap.Rows {
		if room.Kind != "room" || room.SessionID != sessionID || room.Hidden {
			continue
		}
		members := make([]session.GroupMember, 0, len(room.MemberBotIDs))
		for _, id := range room.MemberBotIDs {
			for _, bot := range snap.Rows {
				if bot.ID == id && bot.Kind != "room" {
					members = append(members, session.GroupMember{ID: bot.ID, Name: bot.Name, Title: bot.Title})
					break
				}
			}
		}
		return true, s.refreshGroupRuntime(ep, room.ID, sessionID, members)
	}
	return false, nil
}

// BeginGroupReply enters group-thread reply mode on draft when threadID exists
// in the active room session's projected group threads.
func (s *Service) BeginGroupReply(sessions *session.Store, draft GroupReplyTarget, id, threadID string) {
	if s == nil || draft == nil || sessions == nil || sessions.SelectedSessionID() != id {
		return
	}
	room, members := s.GroupMembers(id)
	if !room {
		return
	}
	for _, thread := range session.ProjectGroupThreads(sessions.Snapshot().Messages, members) {
		if thread.ID == threadID {
			draft.BeginGroupReply(threadID)
			return
		}
	}
}

// Read-only runtime refresh is bounded to the visibly selected canonical room.
// A scope change makes both the pending result and its repaint timer obsolete.
func (s *Service) refreshGroupRuntime(ep harness.Endpoint, roomID, sessionID string, members []session.GroupMember) []session.GroupMember {
	host := s.safeHost()
	sessions := host.Sessions()
	win := host.Window()
	client := host.CurrentClient()
	if sessions == nil || client == nil || win == nil || client.Endpoint.Origin != ep.Origin || client.Endpoint.Cookie != ep.Cookie {
		return members
	}
	snap := sessions.SelectionSummary()
	if snap.SelectedID != sessionID {
		return members
	}
	running := sessions.PromptBusy() || snap.SelectedRunning
	scope := fmt.Sprintf("%p:%s:%s:%s:%d", client, ep.Origin, ep.Cookie, sessionID, snap.ComposerHostGeneration)
	live := &s.assistantGroupLive
	live.mu.Lock()
	live.rendered, live.visible = true, true
	if live.scope != scope {
		live.stopLocked()
		live.rendered, live.visible = true, true
		live.scope = scope
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
	rpc := host.PluginRPCClient(ep)
	current := func() bool {
		live.mu.Lock()
		visible := live.visible && live.scope == scope && live.generation == generation
		live.mu.Unlock()
		if !visible {
			return false
		}
		if host.CurrentClient() != client {
			return false
		}
		bootEp, bootState, _ := host.BootEndpoint()
		cfg := host.Config()
		valid := host.Sessions() == sessions && bootState == harness.Ready && bootEp == ep && cfg != nil && cfg.Get().DshbotEnabled
		if !valid {
			return false
		}
		selected := sessions.SelectionSummary()
		return selected.SelectedID == sessionID && selected.ComposerHostGeneration == snap.ComposerHostGeneration
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
					host.WindowUpdate(win, func() {})
				}
			})
		}
		live.mu.Unlock()
		host.WindowUpdate(win, func() {})
	}()
	return members
}

// Original BotPage polls every known room while mounted, including its task
// and routine views. The native assistant mounts those blocks together.
func (s *Service) contactRoomRuntimes(ep harness.Endpoint, catalog pluginrpc.DshbotContactCatalog) map[string]contactRoomRuntime {
	host := s.safeHost()
	client := s.roomClient(ep)
	asst := s.AssistantStore()
	sessions := host.Sessions()
	win := host.Window()
	if client == nil || asst == nil || sessions == nil || win == nil {
		return nil
	}
	selectedID := sessions.SelectedSessionID()
	var identities []string
	targets := map[string]string{}
	interval := 5 * time.Second
	for _, row := range catalog.Items {
		if row.Kind != "room" || row.ID == "" || row.SessionID == "" {
			continue
		}
		targets[row.ID] = row.SessionID
		identities = append(identities, row.ID+":"+row.SessionID)
		if row.SessionID == selectedID {
			interval = 1500 * time.Millisecond
		}
	}
	if len(targets) == 0 {
		return nil
	}
	sort.Strings(identities)
	scope := fmt.Sprintf("%p:%s:%s:%s:%s", client, ep.Origin, ep.Cookie, selectedID, strings.Join(identities, "\x00"))
	live := &s.assistantGroupLive.contacts
	live.mu.Lock()
	if live.scope != scope {
		live.stopLocked()
		live.scope = scope
		live.rows = map[string]contactRoomRuntime{}
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
	out := make(map[string]contactRoomRuntime, len(live.rows))
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
		if !visible || s.roomClient(ep) != client {
			return false
		}
		same := s.AssistantStore() == asst && host.Sessions() == sessions
		return same && sessions.SelectedSessionID() == selectedID
	}
	rpc := host.PluginRPCClient(ep)
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
				row.err = i18n.T(i18n.KeyCmdRoomIdentityChanged)
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
				host.WindowUpdate(win, func() {})
			}
		})
		live.mu.Unlock()
		host.WindowUpdate(win, func() {})
	}()
	return out
}

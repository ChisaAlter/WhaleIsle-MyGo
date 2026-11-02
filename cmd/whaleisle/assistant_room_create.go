package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) assistantRoomClient(ep harness.Endpoint) *dsh.Client {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if !s.assistantDshbotCurrent(ep, client) {
		return nil
	}
	s.mu.Lock()
	active := s.assistantOpen && !s.quitting && !s.launcherOpen && !s.settingsOpen && !s.usageOpen && !s.imOpen && !s.marketOpen && !s.scheduleOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
	asst, win := s.assistant, s.win
	s.mu.Unlock()
	if !active || asst == nil || win == nil || !win.IsVisible() || win.IsMinimized() || asst.Snapshot().Mode != assiststore.ModeDshbot {
		return nil
	}
	return client
}
func (s *mainState) assistantRoomCreateModel(ep harness.Endpoint) assistantui.RoomCreateModel {
	client := s.assistantRoomClient(ep)
	scope, binding := "", ""
	if client != nil {
		digest := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie))
		scope = fmt.Sprintf("%x", digest)
		binding = fmt.Sprintf("%p", client)
	}
	controls := &s.assistantRoomCreator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	s.mu.Lock()
	workspaceSessions := s.sessions
	s.mu.Unlock()
	var workspaceOptions []session.WorkspaceRow
	workspacesKnown := false
	if workspaceSessions != nil {
		workspaceOptions, workspacesKnown = workspaceSessions.CachedWorkspaceChoices(client)
	}
	current := func() bool {
		return client != nil && s.assistantRoomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	read := func(ctx context.Context) (pluginrpc.DshbotRoomCatalog, bool, error) {
		var authority struct {
			Writable bool `json:"writable"`
		}
		if !current() {
			return pluginrpc.DshbotRoomCatalog{}, false, fmt.Errorf("Host or account changed")
		}
		authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !current() {
			return pluginrpc.DshbotRoomCatalog{}, false, fmt.Errorf("Host or account changed")
		}
		catalog, err := s.pluginRPCClient(ep).DshbotRoomCatalog(ctx)
		return catalog, authorityErr == nil && authority.Writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.updateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, catalog, writable, err) {
				s.updateAssistantTasks()
			}
		}()
	}
	openCreated := func(item pluginrpc.DshbotRoomItem, cwd string) {
		if !current() || item.ID == "" {
			return
		}
		s.openAssistantCreatedRoom(ep, client, current, item, cwd)
	}
	return assistantui.RoomCreateModel{State: snapshot, HostLabel: ep.Origin, Workspaces: workspaceOptions, WorkspacesKnown: workspacesKnown, OnRefresh: refresh,
		OnStart: func() {
			if current() {
				s.mu.Lock()
				sessions := s.sessions
				s.mu.Unlock()
				cwd := ""
				if sessions != nil {
					cwd = sessions.ScratchCwd()
				}
				controls.Start(scope, cwd)
				s.updateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.updateAssistantTasks()
			}
		},
		OnMember: func(id string) {
			if current() {
				controls.ToggleMember(scope, id)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				controls.Discard(scope)
				s.updateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				controls.Dismiss(scope)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.updateAssistantTasks()
			}
		},
		OnAdopt: func(id string) {
			if current() && controls.Adopt(scope, id) {
				s.updateAssistantTasks()
			}
		},
		OnOpen: func() { openCreated(snapshot.Created, snapshot.CreatedScratchCwd) },
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotRoomCatalog(cmd.Expected, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotCatalogReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// run, member-session creation, auto-retry or fresh revision substitution.
				result, createErr := s.pluginRPCClient(ep).DshbotCreateRoom(ctx, cmd)
				controls.Finish(ticket, result, createErr)
				if createErr == nil && current() {
					s.mu.Lock()
					roster := s.dshbotRoster
					s.mu.Unlock()
					if roster != nil {
						items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
						for i, row := range result.Catalog.Items {
							items[i] = row.DshbotCatalogItem
						}
						roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Revision: result.Catalog.Revision, Value: pluginrpc.DshbotCatalogValue{Items: items}}}, false, ep.Origin)
					}
					openCreated(result.Item, cmd.Draft.ScratchCwd)
				}
				s.updateAssistantTasks()
			}()
		},
	}
}

// Observe the latest state after rendering, including navigation performed by
// an earlier callback in this frame. Suspends retire old view callbacks, while
// a dispatched create still settles only its originating account's attempt.
func (s *mainState) observeAssistantRoomCreator(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantRoomCreator.Suspend()
	}
}

// Opening is separate from creation and is bound to the confirmed client and
// view generation. No replacement client or newly opened view can adopt it.
// A failed/stale open never resends bot/create or discards the created identity.
func (s *mainState) openAssistantCreatedRoom(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotRoomItem, cwd string) {
	if client == nil || current == nil || !current() || !s.assistantDshbotCurrent(ep, client) {
		return
	}
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	if asst == nil || sessions == nil || win == nil || s.assistantPluginBusy {
		s.mu.Unlock()
		return
	}
	s.assistantPluginBusy = true
	label := i18n.T(i18n.KeyCmdCreatedRoomOpenCanonicalSession)
	s.assistantPluginDetail = label + "…"
	s.assistantLastAct = s.assistantPluginDetail
	s.mu.Unlock()
	previous := sessions.SelectedSessionID()
	asst.SelectDshbot(expected.ID, ep.Origin)
	live := func() bool {
		if !current() || !s.assistantDshbotCurrent(ep, client) {
			return false
		}
		s.mu.Lock()
		same := s.assistant == asst && s.sessions == sessions && s.win == win
		s.mu.Unlock()
		selected := asst.Snapshot()
		return same && selected.DshbotID == expected.ID && selected.DshbotOrigin == ep.Origin && sessions.SelectedSessionID() == previous
	}
	s.updateAssistantTasks()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		rpc := s.pluginRPCClient(ep) // immutable reviewed endpoint/account, never rebound
		open := func() (string, error) {
			if !live() {
				return "", fmt.Errorf("created room opening expired before catalog read")
			}
			catalog, err := rpc.DshbotDescribeTyped(ctx, expected.ID)
			if err != nil {
				return "", err
			}
			if !live() {
				return "", fmt.Errorf("created room opening expired after catalog read")
			}
			var item pluginrpc.DshbotCatalogItem
			for _, row := range catalog.View.Value.Items {
				if row.ID == expected.ID {
					item = row
					break
				}
			}
			if item.ID != expected.ID || item.Kind != "room" || item.Hidden || item.SessionID != expected.SessionID || item.WorkspaceID != expected.WorkspaceID {
				return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdCreatedRoomIdentityChanged))
			}
			if !live() {
				return "", fmt.Errorf("created room opening expired before canonical open")
			}
			sid, err := rpc.DshbotOpen(ctx, item, catalog.View.Revision, cwd)
			if err != nil {
				return "", fmt.Errorf(i18n.T(i18n.KeyCmdCreatedRoomOpenFailed), err)
			}
			if !live() {
				return "", fmt.Errorf("created room opening expired after canonical receipt")
			}
			return sid, nil
		}
		sid, err := open()
		// Deliver on the UI thread, so close/hide/mode/selection actions cannot
		// interleave between the final view check and local session navigation.
		mygo.RunOnMain(func() {
			valid := live()
			s.mu.Lock()
			s.assistantPluginBusy = false // this worker exclusively owns the busy flag
			if valid {
				if err != nil {
					s.assistantPluginDetail = label + ": " + err.Error()
				} else {
					s.assistantPluginDetail = label + ": " + sid
				}
				s.assistantLastAct = s.assistantPluginDetail
			}
			s.mu.Unlock()
			if valid && err == nil && live() {
				asst.SetSessionID(sid)
				s.revokePhoneGit()
				sessions.Select(sid)
				s.assistantRoomCreator.Suspend()
				s.mu.Lock()
				s.assistantConversationGeneration++
				s.assistantOpen = false
				s.mu.Unlock()
				go func() { sessions.FollowSelected(context.Background()); sessions.RefreshList(context.Background()) }()
			}
			s.updateAssistantTasks()
		})
	}()
}

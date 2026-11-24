package assistantctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *Service) roomClient(ep harness.Endpoint) *dsh.Client {
	if s == nil {
		return nil
	}
	host := s.safeHost()
	client := host.CurrentClient()
	if !s.DshbotCurrent(ep, client) {
		return nil
	}
	blocking := host.OtherSurfaceBlocking()
	win := host.Window()
	winSt := host.WindowStatus(win)
	asst := s.AssistantStore()
	s.mu.Lock()
	open := s.open
	s.mu.Unlock()
	if !open || blocking || asst == nil || !winSt.Exists || !winSt.Visible || winSt.Minimized {
		return nil
	}
	if asst.Snapshot().Value.Mode != "dshbot" {
		return nil
	}
	return client
}

func (s *Service) isRoomInRoster(ep harness.Endpoint, id string) bool {
	roster := s.DshbotRoster()
	if roster == nil || id == "" {
		return false
	}
	cached := roster.Snapshot().Value
	if cached.Origin != ep.Origin {
		return false
	}
	for _, row := range cached.Rows {
		if row.ID == id && row.Kind == "room" {
			return true
		}
	}
	return false
}

// RoomCreateModel builds the room creation draft/review model.
func (s *Service) RoomCreateModel(ep harness.Endpoint) assistantui.RoomCreateModel {
	if s == nil {
		return assistantui.RoomCreateModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	scope, binding := "", ""
	if client != nil {
		scope = endpointScope(ep.Origin, ep.Cookie)
		binding = clientBinding(client)
	}
	controls := &s.assistantRoomCreator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope).Value
	workspaceSessions := host.Sessions()
	var workspaceOptions []session.WorkspaceRow
	workspacesKnown := false
	if workspaceSessions != nil {
		workspaceOptions, workspacesKnown = workspaceSessions.CachedWorkspaceChoices(client)
	}
	current := func() bool {
		return client != nil && s.roomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	rpc := host.PluginRPCClient(ep)
	read := func(ctx context.Context) (pluginrpc.DshbotRoomCatalog, bool, error) {
		writable, authErr := checkWriteAuthority(ctx, client, current, "Host or account changed")
		if authErr != nil && !current() {
			return pluginrpc.DshbotRoomCatalog{}, false, authErr
		}
		catalog, err := rpc.DshbotRoomCatalog(ctx)
		return catalog, authErr == nil && writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, catalog, writable, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	openCreated := func(item pluginrpc.DshbotRoomItem, cwd string) {
		if !current() || item.ID == "" {
			return
		}
		s.openCreatedRoom(ep, client, current, item, cwd)
	}
	return assistantui.RoomCreateModel{
		State:           snapshot,
		HostLabel:       ep.Origin,
		Workspaces:      workspaceOptions,
		WorkspacesKnown: workspacesKnown,
		OnRefresh:       refresh,
		OnStart: func() {
			if current() {
				sessions := host.Sessions()
				cwd := ""
				if sessions != nil {
					cwd = sessions.ScratchCwd()
				}
				controls.Start(scope, cwd)
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnMember: func(id string) {
			if current() {
				controls.ToggleMember(scope, id)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				controls.Discard(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				controls.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func(id string) {
			if current() && controls.Adopt(scope, id) {
				s.UpdateAssistantTasks()
			}
		},
		OnOpen: func() { openCreated(snapshot.Created, snapshot.CreatedScratchCwd) },
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
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
					s.UpdateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// run, member-session creation, auto-retry or fresh revision substitution.
				result, createErr := rpc.DshbotCreateRoom(ctx, cmd)
				controls.Finish(ticket, result, createErr)
				if createErr == nil && current() {
					items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
					for i, row := range result.Catalog.Items {
						items[i] = row.DshbotCatalogItem
					}
					s.updateRosterItems(ep.Origin, result.Catalog.Revision, items)
					openCreated(result.Item, cmd.Draft.ScratchCwd)
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveRoomCreator suspends the room creator when unrendered or offline.
func (s *Service) ObserveRoomCreator(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantRoomCreator.Suspend()
	}
}

// Opening is separate from creation and is bound to the confirmed client and
// view generation. No replacement client or newly opened view can adopt it.
// A failed/stale open never resends bot/create or discards the created identity.
func (s *Service) openCreatedRoom(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotRoomItem, cwd string) {
	if client == nil || current == nil || !current() || !s.DshbotCurrent(ep, client) {
		return
	}
	host := s.safeHost()
	asst := s.AssistantStore()
	sessions := host.Sessions()
	win := host.Window()
	if asst == nil || sessions == nil || win == nil {
		return
	}
	label := i18n.T(i18n.KeyCmdCreatedRoomOpenCanonicalSession)
	s.mu.Lock()
	if s.pluginBusy {
		s.mu.Unlock()
		return
	}
	s.pluginBusy = true
	s.pluginDetail = label + "…"
	s.lastAct = s.pluginDetail
	s.mu.Unlock()

	previous := sessions.SelectedSessionID()
	asst.SelectDshbot(expected.ID, ep.Origin)
	live := func() bool {
		if !current() || !s.DshbotCurrent(ep, client) {
			return false
		}
		same := s.AssistantStore() == asst && host.Sessions() == sessions && host.Window() == win
		selected := asst.Snapshot().Value
		return same && selected.DshbotID == expected.ID && selected.DshbotOrigin == ep.Origin && sessions.SelectedSessionID() == previous
	}
	s.UpdateAssistantTasks()
	rpc := host.PluginRPCClient(ep) // immutable reviewed endpoint/account, never rebound
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
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
		host.RunOnMain(func() {
			valid := live()
			s.mu.Lock()
			s.pluginBusy = false // this worker exclusively owns the busy flag
			if valid {
				if err != nil {
					s.pluginDetail = label + ": " + err.Error()
				} else {
					s.pluginDetail = label + ": " + sid
				}
				s.lastAct = s.pluginDetail
			}
			s.mu.Unlock()
			if valid && err == nil && live() {
				asst.SetSessionID(sid)
				host.RevokePhoneGit()
				sessions.Select(sid)
				s.assistantRoomCreator.Suspend()
				s.SetOpen(false)
				host.RevealMainConversation()
				go func() {
					sessions.FollowSelected(context.Background())
					sessions.RefreshList(context.Background())
				}()
			}
			s.UpdateAssistantTasks()
		})
	}()
}

// RoomEditModel builds the group room member/profile editor model.
func (s *Service) RoomEditModel(ep harness.Endpoint, id string) assistantui.RoomEditModel {
	if s == nil {
		return assistantui.RoomEditModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	asst := s.AssistantStore()
	isRoom := s.isRoomInRoster(ep, id)
	scope, binding := "", ""
	if client != nil && asst != nil && isRoom {
		scope = endpointScope(ep.Origin, ep.Cookie, id)
		binding = clientBinding(client)
	}
	controls := &s.assistantRoomEditor
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope).Value
	current := func() bool {
		if client == nil || asst == nil || scope == "" || s.roomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		same := s.AssistantStore() == asst
		selected := asst.Snapshot().Value
		return same && selected.DshbotID == id && selected.DshbotOrigin == ep.Origin
	}
	rpc := host.PluginRPCClient(ep)
	read := func(ctx context.Context) (pluginrpc.DshbotRoomEditStatus, bool, error) {
		writable, authErr := checkWriteAuthority(ctx, client, current, "Host, account or selected room changed")
		if authErr != nil && !current() {
			return pluginrpc.DshbotRoomEditStatus{}, false, authErr
		}
		status, err := rpc.DshbotRoomEdit(ctx, id)
		if !current() {
			return pluginrpc.DshbotRoomEditStatus{}, false, fmt.Errorf("room view retired")
		}
		return status, authErr == nil && writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			status, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, status, writable, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	change := func(fn func()) {
		if current() {
			fn()
			s.UpdateAssistantTasks()
		}
	}
	return assistantui.RoomEditModel{
		State:         snapshot,
		HostLabel:     ep.Origin,
		OnRefresh:     refresh,
		OnStart:       func() { change(func() { controls.Start(scope) }) },
		OnChange:      func(field, value string) { change(func() { controls.Change(scope, field, value) }) },
		OnMember:      func(member string) { change(func() { controls.ToggleMember(scope, member) }) },
		OnDiscard:     func() { change(func() { controls.Discard(scope) }) },
		OnReview:      func() { change(func() { controls.Prepare(scope) }) },
		OnDismiss:     func() { change(func() { controls.Dismiss(scope) }) },
		OnAcknowledge: func(ack bool) { change(func() { controls.Acknowledge(scope, snapshot.Review, ack) }) },
		OnAdopt:       func() { change(func() { controls.Adopt(scope) }) },
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotRoomEdit(cmd.Expected, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomProfileReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				// One exact bot/update. Never rebase this reviewed payload, retry an unknown
				// result, create/open a session, prompt participants, or invoke run/stop.
				result, saveErr := rpc.DshbotSaveRoomEdit(ctx, cmd)
				controls.Finish(ticket, result, saveErr)
				if saveErr == nil && current() {
					items := make([]pluginrpc.DshbotCatalogItem, len(result.Status.Catalog.Items))
					for i, row := range result.Status.Catalog.Items {
						items[i] = row.DshbotCatalogItem
					}
					s.updateRosterItems(ep.Origin, result.Status.Catalog.Revision, items)
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveRoomEditor suspends the room editor when unrendered or offline.
func (s *Service) ObserveRoomEditor(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantRoomEditor.Suspend()
	}
}

// RoomRuntimeModel builds the room member runtime inspector and member-session opener.
func (s *Service) RoomRuntimeModel(ep harness.Endpoint, id string) assistantui.RoomRuntimeModel {
	if s == nil {
		return assistantui.RoomRuntimeModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	asst := s.AssistantStore()
	sessions := host.Sessions()
	isRoom := s.isRoomInRoster(ep, id)
	scope := ""
	if client != nil && asst != nil && sessions != nil && isRoom {
		scope = endpointScope(clientBinding(client), ep.Origin, ep.Cookie, id)
	}
	reader := &s.assistantRoomRuntime
	reader.Bind(scope)
	snapshot := reader.Snapshot().Value
	current := func() bool {
		if scope == "" || client == nil || s.roomClient(ep) != client || !reader.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		same := s.AssistantStore() == asst && host.Sessions() == sessions
		selected := asst.Snapshot().Value
		return same && selected.DshbotID == id && selected.DshbotOrigin == ep.Origin
	}
	rpc := host.PluginRPCClient(ep)
	return assistantui.RoomRuntimeModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !current() {
				return
			}
			ticket, ok := reader.BeginRead(scope, snapshot.Generation)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				if !current() || !reader.Current(ticket) {
					if reader.FinishRead(ticket, pluginrpc.DshbotRoomRuntime{}, fmt.Errorf("runtime read expired before dispatch")) {
						s.UpdateAssistantTasks()
					}
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				data, err := rpc.DshbotRoomRuntime(ctx, id)
				if current() && reader.FinishRead(ticket, data, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnOpen: func(memberSessionID string) {
			if !current() {
				return
			}
			previous := sessions.SelectedSessionID()
			ticket, expected, ok := reader.BeginOpen(scope, snapshot.Generation, memberSessionID)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			original, err := pluginrpc.DshbotRoomRuntimeTarget(expected, memberSessionID)
			if err != nil {
				reader.FinishOpen(ticket, err)
				s.UpdateAssistantTasks()
				return
			}
			live := func() bool { return current() && reader.Current(ticket) && sessions.SelectedSessionID() == previous }
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				verify := func() error {
					if !live() {
						return fmt.Errorf("member opening expired")
					}
					fresh, err := rpc.DshbotRoomRuntime(ctx, id)
					if err != nil {
						return err
					}
					if !live() {
						return fmt.Errorf("room view changed during runtime read")
					}
					member, err := pluginrpc.DshbotRoomRuntimeTarget(fresh, memberSessionID)
					if err != nil {
						return err
					}
					if fresh.Room.ID != expected.Room.ID || fresh.Room.SessionID != expected.Room.SessionID || member.BotID != original.BotID || member.RoomSessionID != original.RoomSessionID {
						return fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomMemberIdentityChanged))
					}
					// An explicit open may observe/resume an existing root session only. The
					// group member is not a personal Bot canonical session or subagent address.
					var listing api.SessionListValue
					if err = client.Call(ctx, "session/list", map[string]any{"_request": api.SessionListRequest{}}, &listing); err != nil {
						return err
					}
					if !live() {
						return fmt.Errorf("Host or session selection changed")
					}
					found := 0
					for _, row := range listing.Items {
						if row.SessionId == memberSessionID {
							if row.ParentSessionId != nil && *row.ParentSessionId != "" {
								return fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomMemberNotRoot))
							}
							found++
						}
					}
					if found != 1 {
						return fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomMemberUnavailable))
					}
					return nil
				}
				openErr := verify()
				host.RunOnMain(func() {
					if !live() {
						if reader.FinishOpen(ticket, fmt.Errorf("Host, room view or selected session changed before navigation")) {
							s.UpdateAssistantTasks()
						}
						return
					}
					if openErr == nil && !sessions.SelectObservedRoomMember(client, previous, memberSessionID) {
						openErr = fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomMemberBindingExpired))
					}
					reader.FinishOpen(ticket, openErr)
					if openErr == nil {
						host.RevokePhoneGit()
						s.assistantRoomCreator.Suspend()
						s.assistantRoomEditor.Suspend()
						reader.Suspend()
						s.SetOpen(false)
						host.RevealMainConversation()
					}
					s.UpdateAssistantTasks()
				})
			}()
		},
	}
}

// ObserveRoomRuntime suspends the room runtime reader when unrendered or offline.
func (s *Service) ObserveRoomRuntime(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantRoomRuntime.Suspend()
	}
}

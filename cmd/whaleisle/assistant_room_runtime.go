package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) assistantRoomRuntimeModel(ep harness.Endpoint, id string) assistantui.RoomRuntimeModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	asst, roster, sessions := s.assistant, s.dshbotRoster, s.sessions
	s.mu.Unlock()
	isRoom := false
	if roster != nil {
		rows := roster.Snapshot()
		if rows.Origin == ep.Origin {
			for _, row := range rows.Rows {
				if row.ID == id && row.Kind == "room" {
					isRoom = true
				}
			}
		}
	}
	scope := ""
	if client != nil && asst != nil && sessions != nil && isRoom {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%p\x00%s\x00%s\x00%s", client, ep.Origin, ep.Cookie, id)))
		scope = fmt.Sprintf("%x", digest)
	}
	reader := &s.assistantRoomRuntime
	reader.Bind(scope)
	snapshot := reader.Snapshot()
	current := func() bool {
		if scope == "" || client == nil || s.assistantRoomClient(ep) != client || !reader.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		s.mu.Lock()
		same := s.assistant == asst && s.sessions == sessions
		s.mu.Unlock()
		selected := asst.Snapshot()
		return same && selected.DshbotID == id && selected.DshbotOrigin == ep.Origin
	}
	rpc := s.pluginRPCClient(ep)
	return assistantui.RoomRuntimeModel{State: snapshot, HostLabel: ep.Origin,
		OnRefresh: func() {
			if !current() {
				return
			}
			ticket, ok := reader.BeginRead(scope, snapshot.Generation)
			if !ok {
				return
			}
			s.updateAssistantTasks()
			go func() {
				if !current() || !reader.Current(ticket) {
					if reader.FinishRead(ticket, pluginrpc.DshbotRoomRuntime{}, fmt.Errorf("runtime read expired before dispatch")) {
						s.updateAssistantTasks()
					}
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				data, err := rpc.DshbotRoomRuntime(ctx, id)
				if current() && reader.FinishRead(ticket, data, err) {
					s.updateAssistantTasks()
				}
			}()
		},
		OnOpen: func(memberSessionID string) {
			if !current() {
				return
			}
			previous := sessions.SelectedSessionID()
			ticket, expected, ok := reader.BeginOpen(scope, snapshot.Generation, memberSessionID)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			original, err := pluginrpc.DshbotRoomRuntimeTarget(expected, memberSessionID)
			if err != nil {
				reader.FinishOpen(ticket, err)
				s.updateAssistantTasks()
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
				mygo.RunOnMain(func() {
					if !live() {
						if reader.FinishOpen(ticket, fmt.Errorf("Host, room view or selected session changed before navigation")) {
							s.updateAssistantTasks()
						}
						return
					}
					if openErr == nil && !sessions.SelectObservedRoomMember(client, previous, memberSessionID) {
						openErr = fmt.Errorf("%s", i18n.T(i18n.KeyCmdRoomMemberBindingExpired))
					}
					reader.FinishOpen(ticket, openErr)
					if openErr == nil {
						s.revokePhoneGit()
						s.assistantRoomCreator.Suspend()
						s.assistantRoomEditor.Suspend()
						reader.Suspend()
						s.mu.Lock()
						s.assistantConversationGeneration++
						s.assistantOpen = false
						s.mu.Unlock()
					}
					s.updateAssistantTasks()
				})
			}()
		},
	}
}
func (s *mainState) observeAssistantRoomRuntime(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantRoomRuntime.Suspend()
	}
}

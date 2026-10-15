package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantRoomEditModel(ep harness.Endpoint, id string) assistantui.RoomEditModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	asst, roster := s.assistant, s.dshbotRoster
	s.mu.Unlock()
	isRoom := false
	if roster != nil {
		cached := roster.Snapshot()
		if cached.Origin == ep.Origin {
			for _, row := range cached.Rows {
				if row.ID == id && row.Kind == "room" {
					isRoom = true
				}
			}
		}
	}
	scope, binding := "", ""
	if client != nil && asst != nil && isRoom {
		digest := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie + "\x00" + id))
		scope = fmt.Sprintf("%x", digest)
		binding = fmt.Sprintf("%p", client)
	}
	controls := &s.assistantRoomEditor
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	current := func() bool {
		if client == nil || asst == nil || scope == "" || s.assistantRoomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		s.mu.Lock()
		same := s.assistant == asst
		s.mu.Unlock()
		selected := asst.Snapshot()
		return same && selected.DshbotID == id && selected.DshbotOrigin == ep.Origin
	}
	rpc := s.pluginRPCClient(ep) // immutable endpoint/account captured with the original client
	read := func(ctx context.Context) (pluginrpc.DshbotRoomEditStatus, bool, error) {
		var authority struct {
			Writable bool `json:"writable"`
		}
		if !current() {
			return pluginrpc.DshbotRoomEditStatus{}, false, fmt.Errorf("Host, account or selected room changed")
		}
		authErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !current() {
			return pluginrpc.DshbotRoomEditStatus{}, false, fmt.Errorf("Host, account or selected room changed")
		}
		status, err := rpc.DshbotRoomEdit(ctx, id)
		if !current() {
			return pluginrpc.DshbotRoomEditStatus{}, false, fmt.Errorf("room view retired")
		}
		return status, authErr == nil && authority.Writable, err
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
			status, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, status, writable, err) {
				s.updateAssistantTasks()
			}
		}()
	}
	change := func(fn func()) {
		if current() {
			fn()
			s.updateAssistantTasks()
		}
	}
	return assistantui.RoomEditModel{State: snapshot, HostLabel: ep.Origin, OnRefresh: refresh,
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
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotRoomEdit(cmd.Expected, fresh)) {
					err = fmt.Errorf("catalog revision, complete profile or authority changed; refresh and review again")
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// One exact bot/update. Never rebase this reviewed payload, retry an unknown
				// result, create/open a session, prompt participants, or invoke run/stop.
				result, saveErr := rpc.DshbotSaveRoomEdit(ctx, cmd)
				controls.Finish(ticket, result, saveErr)
				if saveErr == nil && current() && roster != nil {
					items := make([]pluginrpc.DshbotCatalogItem, len(result.Status.Catalog.Items))
					for i, row := range result.Status.Catalog.Items {
						items[i] = row.DshbotCatalogItem
					}
					roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Revision: result.Status.Catalog.Revision, Value: pluginrpc.DshbotCatalogValue{Items: items}}}, false, ep.Origin)
				}
				s.updateAssistantTasks()
			}()
		},
	}
}
func (s *mainState) observeAssistantRoomEditor(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantRoomEditor.Suspend()
	}
}

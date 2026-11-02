package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) bindBotCapabilityModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Capabilities
	snapshot := controls.Snapshot(k)
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		var status pluginrpc.DshbotCapabilityStatus
		client := s.assistantRoutineClient(ep, k.BotID)
		if client == nil || !live() {
			return status, false, fmt.Errorf("Bot identity changed")
		}
		var authority struct {
			Writable bool `json:"writable"`
		}
		authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !live() {
			return status, false, fmt.Errorf("Bot identity changed")
		}
		value, err := s.pluginRPCClient(ep).DshbotCapabilities(ctx, k.BotID)
		return value, authorityErr == nil && authority.Writable, err
	}
	m.Capabilities = assistantui.BotCapabilityModel{State: snapshot, HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.updateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				value, writable, err := read(ctx, live)
				if current() && controls.FinishLoad(ticket, value, writable, err) {
					s.updateAssistantTasks()
				}
			}()
		},
		OnChange: func(group, field, value string) {
			if viewCurrent() {
				controls.Change(k, group, field, value)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.updateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.updateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				var out pluginrpc.DshbotCapabilityReceipt
				if err == nil && (!writable || !pluginrpc.SameDshbotCapabilities(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotCapabilityReviewChanged))
				}
				if err == nil && live() {
					out, err = s.pluginRPCClient(ep).DshbotSaveCapabilities(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("view changed before capability save")
				}
				if current() && controls.FinishSave(ticket, out, err) {
					s.updateAssistantTasks()
				}
			}()
		},
	}
}

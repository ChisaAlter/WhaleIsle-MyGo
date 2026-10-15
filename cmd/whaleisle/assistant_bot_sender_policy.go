package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) bindBotSenderPolicyModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.SenderPolicy
	client := s.assistantRoutineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.assistantRoomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k)
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.pluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotSenderPolicyStatus, bool, error) {
		var out pluginrpc.DshbotSenderPolicyStatus
		var auth struct {
			Writable bool `json:"writable"`
		}
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		ae := client.Call(ctx, "settings/describe", nil, &auth)
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		var err error
		out, err = rpc.DshbotSenderPolicy(ctx, k.BotID)
		if err != nil {
			return out, false, err
		}
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		return out, ae == nil && auth.Writable, nil
	}
	m.SenderPolicy = assistantui.BotSenderPolicyModel{State: snapshot, HostLabel: ep.Origin,
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
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.updateAssistantTasks()
				}
			}()
		},
		OnChange: func(v pluginrpc.DshbotSenderPolicyValues) {
			if viewCurrent() {
				controls.SetDraft(k, v)
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
		OnExpansionAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.AcknowledgeExpansion(k, snapshot.Review, ack)
				s.updateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
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
				if err == nil && (!writable || !pluginrpc.SameDshbotSenderPolicy(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("reviewed profile, peer identities, shared rooms or authority changed; refresh and review")
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Exactly one full-profile update. Never send messages/tasks, rebase revision,
				// or retry an unknown outcome. Host bot/update may request objective sync.
				receipt, saveErr := rpc.DshbotSaveSenderPolicy(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.updateAssistantTasks()
			}()
		},
	}
}

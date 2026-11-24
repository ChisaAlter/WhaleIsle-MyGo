package assistantctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *Service) routineClient(ep harness.Endpoint, botID string) *dsh.Client {
	s.mu.Lock()
	active := s.open
	s.mu.Unlock()
	if !active {
		return nil
	}
	return s.taskClient(ep, botID)
}

// RoutinesModel builds the Bot routines, trigger, memory, capabilities, objective,
// route, avatar, notify, sender-policy, and text-profile composite model.
func (s *Service) RoutinesModel(ep harness.Endpoint, botID string) assistantui.RoutineModel {
	if s == nil {
		return assistantui.RoutineModel{}
	}
	host := s.safeHost()
	client := s.routineClient(ep, botID)
	tScope := taskScope(ep, client, botID)
	scope := dshbotstore.RoutineScope{HostKey: tScope.HostKey, BotID: tScope.BotID}
	s.assistantRoutines.Bind(scope)
	binding := s.assistantRoutines.Binding(scope)
	current := func() bool {
		return client != nil && s.assistantRoutines.Bound(scope, binding) && s.routineClient(ep, botID) == client
	}
	rpc := host.PluginRPCClient(ep)
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := s.assistantRoutines.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var view *pluginrpc.DshbotRoutineView
			var ids []string
			var err, missedErr error
			if current() && s.assistantRoutines.Current(ticket) {
				view, err = rpc.DshbotRoutines(ctx, botID)
				if err == nil && current() && s.assistantRoutines.Current(ticket) {
					ids, missedErr = rpc.DshbotMissedRoutines(ctx)
				}
			} else {
				err = fmt.Errorf("Host or Bot selection changed")
			}
			if !current() {
				return
			}
			if s.assistantRoutines.FinishLoad(ticket, view, ids, missedErr, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	snapshot := s.assistantRoutines.Snapshot(scope).Value
	model := assistantui.RoutineModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: refresh,
		OnSelect: func(id string) {
			if current() {
				s.assistantRoutines.Select(scope, id)
				s.UpdateAssistantTasks()
			}
		},
		OnEdit: func() {
			if current() {
				s.assistantRoutines.StartEdit(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				s.assistantRoutines.Edit(scope, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				s.assistantRoutines.DiscardEdit(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func(action string) {
			if current() {
				s.assistantRoutines.Prepare(scope, action)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantRoutines.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledgeWatch: func(ack bool) {
			if current() {
				s.assistantRoutines.AcknowledgeWatch(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantRoutines.BeginAction(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				var view *pluginrpc.DshbotRoutineView
				var err error
				// Dispatch exactly the reviewed operation once. Never open a Bot session,
				// generate a token, retry run/toggle, or submit a chat prompt here.
				if current() && s.assistantRoutines.Current(ticket) {
					view, err = rpc.DshbotControlRoutine(ctx, command)
				} else {
					err = fmt.Errorf("Host or Bot selection changed before the action")
				}
				if !current() {
					return
				}
				if s.assistantRoutines.FinishAction(ticket, view, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
	s.bindRoutineTriggerModel(&model, ep, scope, current)
	s.bindBotMemoryModel(&model, ep, scope, current)
	s.bindBotMemoryReviewModel(&model, ep, scope, current)
	s.bindBotCapabilityModel(&model, ep, scope, current)
	s.bindBotObjectiveModel(&model, ep, scope, current)
	s.bindBotObjectiveConfigModel(&model, ep, scope, current)
	s.bindBotRouteModel(&model, ep, scope, current)
	s.bindBotAvatarModel(&model, ep, scope, current)
	s.bindBotNotifyModel(&model, ep, scope, current)
	s.bindBotSenderPolicyModel(&model, ep, scope, current)
	s.bindBotTextProfileModel(&model, ep, scope, current)
	return model
}

// RoutineCreateModel builds the routine creation draft/review model for a Bot.
func (s *Service) RoutineCreateModel(ep harness.Endpoint, botID string, source dshbotstore.RoutineSnapshot, refresh func()) assistantui.RoutineCreateModel {
	if s == nil {
		return assistantui.RoutineCreateModel{}
	}
	host := s.safeHost()
	client := s.routineClient(ep, botID)
	var scope dshbotstore.RoutineScope
	if client != nil {
		// Stable across client reconnects, isolated by Host/account/Bot. Keep pending
		// creates after navigation; never display or persist the cookie/fingerprint.
		scope = dshbotstore.RoutineScope{
			HostKey: endpointScope(ep.Origin, ep.Cookie),
			BotID:   botID,
		}
	}
	binding := s.assistantRoutines.Binding(source.Scope)
	s.assistantRoutineCreate.Sync(scope, binding, source)
	snapshot := s.assistantRoutineCreate.Snapshot(scope).Value
	current := func() bool {
		return client != nil && s.routineClient(ep, botID) == client && s.assistantRoutines.Bound(source.Scope, binding)
	}
	rpc := host.PluginRPCClient(ep)
	return assistantui.RoutineCreateModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		BotName:   source.BotName,
		Routines:  source.Routines,
		OnAcknowledgeWatch: func(ack bool) {
			if current() {
				s.assistantRoutineCreate.AcknowledgeWatch(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnStart: func() {
			if current() {
				s.assistantRoutineCreate.Start(scope, time.Now().Location().String())
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				s.assistantRoutineCreate.Change(scope, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				s.assistantRoutineCreate.Discard(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnRebase: func() {
			if current() {
				s.assistantRoutineCreate.Rebase(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				s.assistantRoutineCreate.Prepare(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantRoutineCreate.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnRefresh: func() {
			if current() && refresh != nil {
				refresh()
			}
		},
		OnAdopt: func(id string) {
			if current() && s.assistantRoutineCreate.Adopt(scope, id) {
				s.assistantRoutines.Select(source.Scope, id)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantRoutineCreate.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				if !current() || !s.assistantRoutineCreate.Current(ticket) || !s.assistantRoutineCreate.MarkSent(ticket) {
					s.assistantRoutineCreate.AbortUnsent(ticket)
					s.UpdateAssistantTasks()
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				result, err := rpc.DshbotCreateRoutine(ctx, command)
				// Finish updates only the originating account/Bot's captured attempt. A
				// different active Host never receives its result, even after navigation.
				s.assistantRoutineCreate.Finish(ticket, result, err)
				if current() && refresh != nil {
					refresh()
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

func (s *Service) bindRoutineTriggerModel(m *assistantui.RoutineModel, ep harness.Endpoint, scope dshbotstore.RoutineScope, current func() bool) {
	if _, err := pluginrpc.DshbotRoutineTriggerURL(ep.Origin, m.State.SelectedID); err != nil {
		m.TriggerUnavailable = i18n.T(i18n.KeyCmdUnsupportedTriggerAddressNoCredentia)
		return
	}
	host := s.safeHost()
	rpc := host.PluginRPCClient(ep)
	sequence := m.State.Trigger.Sequence
	validView := func() bool {
		if current() {
			return true
		}
		s.assistantRoutines.CloseTrigger(scope, sequence)
		return false
	}
	m.OnPrepareTrigger = func() {
		if validView() {
			s.assistantRoutines.PrepareTrigger(scope)
			s.UpdateAssistantTasks()
		}
	}
	m.OnAcknowledgeTrigger = func(ack bool) {
		if validView() {
			s.assistantRoutines.AcknowledgeTrigger(scope, sequence, ack)
			s.UpdateAssistantTasks()
		}
	}
	m.OnCloseTrigger = func() {
		if validView() {
			s.assistantRoutines.CloseTrigger(scope, sequence)
			s.UpdateAssistantTasks()
		}
	}
	m.OnConfirmTrigger = func() {
		if !validView() {
			return
		}
		ticket, ok := s.assistantRoutines.BeginTrigger(scope, sequence)
		s.UpdateAssistantTasks()
		if !ok {
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var token []byte
			var err error
			if validView() && s.assistantRoutines.CurrentTrigger(ticket) {
				token, err = rpc.DshbotObtainRoutineTriggerToken(ctx)
			} else {
				err = fmt.Errorf("trigger view changed before dispatch")
			}
			defer clear(token)
			if !validView() {
				return
			}
			if s.assistantRoutines.FinishTrigger(ticket, token, err) && validView() {
				s.UpdateAssistantTasks()
			}
		}()
	}
	m.TriggerMaterial = func() (string, string) {
		if !validView() {
			return "", ""
		}
		address, header, err := s.assistantRoutines.TriggerMaterial(scope, sequence, ep.Origin)
		if err != nil {
			return "", ""
		}
		return address, header
	}
	m.OnCopyTrigger = func(part string) {
		if !validView() {
			return
		}
		address, header, err := s.assistantRoutines.TriggerMaterial(scope, sequence, ep.Origin)
		if err != nil {
			return
		}
		// Only an explicit user click calls clipboard. No automatic execution,
		// clipboard polling, logging, file write or background credential refresh.
		switch part {
		case "address":
			host.WriteClipboard(address)
		case "header":
			host.WriteClipboard(header)
		}
	}
}

package main

import (
	"context"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantRoutineClient(ep harness.Endpoint, botID string) *dsh.Client {
	s.mu.Lock()
	active := s.assistantOpen
	s.mu.Unlock()
	if !active {
		return nil
	}
	return s.assistantTaskClient(ep, botID)
}
func (s *mainState) assistantRoutinesModel(ep harness.Endpoint, botID string) assistantui.RoutineModel {
	client := s.assistantRoutineClient(ep, botID)
	taskScope := assistantTaskScope(ep, client, botID)
	scope := dshbotstore.RoutineScope{HostKey: taskScope.HostKey, BotID: taskScope.BotID}
	s.assistantRoutines.Bind(scope)
	binding := s.assistantRoutines.Binding(scope)
	current := func() bool {
		return client != nil && s.assistantRoutines.Bound(scope, binding) && s.assistantRoutineClient(ep, botID) == client
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := s.assistantRoutines.BeginLoad(scope)
		if !ok {
			return
		}
		s.updateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var view *pluginrpc.DshbotRoutineView
			var ids []string
			var err, missedErr error
			if current() && s.assistantRoutines.Current(ticket) {
				rpc := s.pluginRPCClient(ep)
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
				s.updateAssistantTasks()
			}
		}()
	}
	snapshot := s.assistantRoutines.Snapshot(scope)
	model := assistantui.RoutineModel{
		State: snapshot, HostLabel: ep.Origin, OnRefresh: refresh,
		OnSelect: func(id string) {
			if current() {
				s.assistantRoutines.Select(scope, id)
				s.updateAssistantTasks()
			}
		},
		OnEdit: func() {
			if current() {
				s.assistantRoutines.StartEdit(scope)
				s.updateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				s.assistantRoutines.Edit(scope, field, value)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				s.assistantRoutines.DiscardEdit(scope)
				s.updateAssistantTasks()
			}
		},
		OnReview: func(action string) {
			if current() {
				s.assistantRoutines.Prepare(scope, action)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantRoutines.Dismiss(scope)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledgeWatch: func(ack bool) {
			if current() {
				s.assistantRoutines.AcknowledgeWatch(scope, snapshot.Review, ack)
				s.updateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantRoutines.BeginAction(scope, snapshot.Review)
			s.updateAssistantTasks()
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
					view, err = s.pluginRPCClient(ep).DshbotControlRoutine(ctx, command)
				} else {
					err = fmt.Errorf("Host or Bot selection changed before the action")
				}
				if !current() {
					return
				}
				if s.assistantRoutines.FinishAction(ticket, view, err) {
					s.updateAssistantTasks()
				}
			}()
		},
	}
	s.bindRoutineTriggerModel(&model, ep, scope, current)
	s.bindBotMemoryModel(&model, ep, scope, current)
	s.bindBotCapabilityModel(&model, ep, scope, current)
	s.bindBotObjectiveModel(&model, ep, scope, current)
	return model
}

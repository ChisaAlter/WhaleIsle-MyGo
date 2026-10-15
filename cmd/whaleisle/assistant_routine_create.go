package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantRoutineCreateModel(ep harness.Endpoint, botID string, source dshbotstore.RoutineSnapshot, refresh func()) assistantui.RoutineCreateModel {
	client := s.assistantRoutineClient(ep, botID)
	var scope dshbotstore.RoutineScope
	if client != nil {
		// Stable across client reconnects, isolated by Host/account/Bot. Keep pending
		// creates after navigation; never display or persist the cookie/fingerprint.
		sum := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie))
		scope = dshbotstore.RoutineScope{HostKey: fmt.Sprintf("%x", sum), BotID: botID}
	}
	binding := s.assistantRoutines.Binding(source.Scope)
	s.assistantRoutineCreator.Sync(scope, binding, source)
	snapshot := s.assistantRoutineCreator.Snapshot(scope)
	current := func() bool {
		return client != nil && s.assistantRoutineClient(ep, botID) == client && s.assistantRoutines.Bound(source.Scope, binding)
	}
	return assistantui.RoutineCreateModel{
		State: snapshot, HostLabel: ep.Origin, BotName: source.BotName,
		OnStart: func() {
			if current() {
				s.assistantRoutineCreator.Start(scope, time.Now().Location().String())
				s.updateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				s.assistantRoutineCreator.Change(scope, field, value)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				s.assistantRoutineCreator.Discard(scope)
				s.updateAssistantTasks()
			}
		},
		OnRebase: func() {
			if current() {
				s.assistantRoutineCreator.Rebase(scope)
				s.updateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				s.assistantRoutineCreator.Prepare(scope)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantRoutineCreator.Dismiss(scope)
				s.updateAssistantTasks()
			}
		},
		OnRefresh: func() {
			if current() && refresh != nil {
				refresh()
			}
		},
		OnAdopt: func(id string) {
			if current() && s.assistantRoutineCreator.Adopt(scope, id) {
				s.assistantRoutines.Select(source.Scope, id)
				s.updateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantRoutineCreator.Begin(scope, snapshot.Review)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				if !current() || !s.assistantRoutineCreator.Current(ticket) || !s.assistantRoutineCreator.MarkSent(ticket) {
					s.assistantRoutineCreator.AbortUnsent(ticket)
					s.updateAssistantTasks()
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				result, err := s.pluginRPCClient(ep).DshbotCreateRoutine(ctx, command)
				// Finish updates only the originating account/Bot's captured attempt. A
				// different active Host never receives its result, even after navigation.
				s.assistantRoutineCreator.Finish(ticket, result, err)
				if current() && refresh != nil {
					refresh()
				}
				s.updateAssistantTasks()
			}()
		},
	}
}

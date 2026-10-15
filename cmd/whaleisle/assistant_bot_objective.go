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

func (s *mainState) bindBotObjectiveModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Objective
	snapshot := controls.Snapshot(k)
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	m.Objective = assistantui.BotObjectiveModel{State: snapshot, HostLabel: ep.Origin,
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
				var status pluginrpc.DshbotObjectiveStatus
				var err error
				if current() && controls.Current(ticket) {
					status, err = s.pluginRPCClient(ep).DshbotObjective(ctx, k.BotID)
				} else {
					err = fmt.Errorf("objective view changed before read")
				}
				if current() && controls.FinishLoad(ticket, status, err) {
					s.updateAssistantTasks()
				}
			}()
		},
		OnReview: func(action string) {
			if viewCurrent() {
				controls.Prepare(k, action)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.updateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginAction(k, snapshot.Review)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				var out pluginrpc.DshbotObjectiveResult
				var err error
				if live() {
					var fresh pluginrpc.DshbotObjectiveStatus
					fresh, err = s.pluginRPCClient(ep).DshbotObjective(ctx, k.BotID)
					if err == nil && !pluginrpc.DshbotObjectivePreflightMatches(cmd, fresh) {
						err = fmt.Errorf("objective changed after review; no action sent")
					}
				} else {
					err = fmt.Errorf("objective view changed before preflight")
				}
				// Final dispatch fence after the read. This cannot add an atomic CAS that
				// the original start/pause endpoints do not implement; the review says so.
				if err == nil && live() {
					out, err = s.pluginRPCClient(ep).DshbotControlObjective(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("objective view changed before dispatch")
				}
				if current() && controls.FinishAction(ticket, out, err) {
					s.updateAssistantTasks()
				}
			}()
		},
	}
}

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

func (s *mainState) bindBotMemoryModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Memory
	snapshot := controls.Snapshot(k)
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	// Check the existing Host settings authority. Unlike catalog mutations,
	// upstream memory/replace checks only the independent track CAS revision.
	readIdentity := func(ctx context.Context, stillCurrent func() bool) (pluginrpc.DshbotCatalogItem, bool, error) {
		var bot pluginrpc.DshbotCatalogItem
		client := s.assistantRoutineClient(ep, k.BotID)
		if client == nil || !stillCurrent() {
			return bot, false, fmt.Errorf("Bot identity changed")
		}
		var authority struct {
			Writable bool `json:"writable"`
		}
		authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !stillCurrent() {
			return bot, false, fmt.Errorf("Bot identity changed")
		}
		view, err := s.pluginRPCClient(ep).DshbotRoutines(ctx, k.BotID)
		if err != nil {
			return bot, false, err
		}
		bot, ok := view.Bot(k.BotID)
		if !ok {
			return bot, false, fmt.Errorf("individual Bot unavailable")
		}
		return bot, authorityErr == nil && authority.Writable, nil
	}
	m.Memory = assistantui.BotMemoryModel{State: snapshot, HostLabel: ep.Origin,
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
				bot, writable, err := readIdentity(ctx, live)
				var value pluginrpc.DshbotMemoryResult
				if err == nil && live() {
					value, err = s.pluginRPCClient(ep).DshbotMemory(ctx, k.BotID)
				} else if err == nil {
					err = fmt.Errorf("Bot identity changed before memory read")
				}
				if current() && controls.FinishLoad(ticket, bot, value, writable, err) {
					s.updateAssistantTasks()
				}
			}()
		},
		OnTrack: func(track string) {
			if viewCurrent() {
				controls.SetTrack(k, track)
				s.updateAssistantTasks()
			}
		},
		OnEdit: func(index int) {
			if viewCurrent() {
				controls.StartEdit(k, index)
				s.updateAssistantTasks()
			}
		},
		OnDelete: func(index int) {
			if viewCurrent() {
				controls.PrepareDelete(k, index)
				s.updateAssistantTasks()
			}
		},
		OnChange: func(text string) {
			if viewCurrent() {
				controls.Change(k, text)
				s.updateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
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
			ticket, command, ok := controls.BeginSave(k, snapshot.Review)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				bot, writable, err := readIdentity(ctx, live)
				var receipt pluginrpc.DshbotMemoryReceipt
				if err == nil && (!writable || bot.SessionID != command.SessionID) {
					err = fmt.Errorf("memory write authority or canonical Bot identity changed")
				}
				if err == nil && live() {
					receipt, err = s.pluginRPCClient(ep).DshbotReplaceMemory(ctx, command)
				} else if err == nil {
					err = fmt.Errorf("Bot identity changed before memory write")
				}
				if current() && controls.FinishSave(ticket, receipt, err) {
					s.updateAssistantTasks()
				}
			}()
		},
	}
}

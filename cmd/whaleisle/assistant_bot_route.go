package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) bindBotRouteModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.Route
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
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotRouteStatus, bool, error) {
		var out pluginrpc.DshbotRouteStatus
		out.Options.WorkspaceLocked = true
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
		out.DshbotCapabilityStatus, err = rpc.DshbotCapabilities(ctx, k.BotID)
		if err != nil {
			return out, false, err
		}
		var raw json.RawMessage
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		if client.Call(ctx, "session/modelCatalog", map[string]any{}, &raw) == nil {
			_ = pluginrpc.DecodeDshbotRouteModels(raw, &out.Options)
		}
		raw = nil
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		if client.Call(ctx, "session/list", map[string]any{"_request": map[string]any{}}, &raw) == nil {
			_ = pluginrpc.DecodeDshbotRouteSession(raw, out.Bot.SessionID, &out.Options)
		}
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		// A dedicated read-only baseline avoids stale cached directories and never
		// creates a workspace, inspects local paths, or resolves a cold Bot session.
		wctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
		defer mux.Close()
		if mux.Dial(wctx) == nil && live() {
			if stream, e := mux.Open(wctx, "workspace/follow", map[string]any{"args": map[string]any{}}); e == nil {
				defer stream.Cancel()
				if baseline, e := stream.Recv(wctx); e == nil {
					_ = pluginrpc.DecodeDshbotRouteWorkspaces(baseline, &out.Options)
				}
			}
		}
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		return out, ae == nil && auth.Writable, nil
	}
	m.Route = assistantui.BotRouteModel{State: snapshot, HostLabel: ep.Origin,
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
		OnChange: func(v pluginrpc.DshbotRouteValues) {
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
				if err == nil && (!writable || !pluginrpc.SameDshbotRoute(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("reviewed profile, model choices, directory/history or authority changed; refresh and review")
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Exactly one full-profile update. Never open, prompt, start, resume, rebase
				// revision or retry an unknown session/catalog result.
				receipt, saveErr := rpc.DshbotSaveRoute(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.updateAssistantTasks()
			}()
		},
	}
}

// Called after each frame as well as direct window/Host hooks. A same-frame
// route change retires old callbacks even if the window is reopened immediately.
func (s *mainState) observeAssistantBotRoute() {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if s.assistantRoomClient(ep) == nil {
		s.assistantRoutines.TextProfile.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.SenderPolicy.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Notify.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Avatar.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Route.Bind(dshbotstore.RoutineScope{})
	}
}

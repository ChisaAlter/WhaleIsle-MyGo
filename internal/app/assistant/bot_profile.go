package assistantctl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *Service) bindBotAvatarModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.Avatar
	client := s.routineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.roomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		return readBotCapabilityStatus(ctx, client, rpc, k.BotID, live)
	}
	m.Avatar = assistantui.BotAvatarModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(v pluginrpc.DshbotAvatarValues) {
			if viewCurrent() {
				controls.SetDraft(k, v)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				if err == nil && (!writable || !pluginrpc.SameDshbotCapabilities(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotAvatarReviewChanged))
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				// Exactly one full-profile update. Never open, prompt, start, resume, rebase
				// revision or retry an unknown session/catalog result.
				receipt, saveErr := rpc.DshbotSaveAvatar(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.UpdateAssistantTasks()
			}()
		},
	}
	s.bindBotAvatarImageActions(&m.Avatar, k, current, viewCurrent)
}

func (s *Service) bindBotAvatarImageActions(m *assistantui.BotAvatarModel, k dshbotstore.RoutineScope, current, viewCurrent func() bool) {
	controls := &s.assistantRoutines.Avatar
	host := s.safeHost()
	m.OnChooseImage = func() {
		if !viewCurrent() {
			return
		}
		ticket, ok := controls.BeginImageImport(k)
		if !ok {
			return
		}
		win := host.Window()
		s.UpdateAssistantTasks()
		go func() {
			live := func() bool { return current() && controls.ImageImportCurrent(ticket) }
			source, err := avatarimage.Select(func() ([]string, error) {
				title := i18n.T(i18n.KeyCmdChooseLocalAvatarImage)
				if ad, ok := host.(hostDepsAdapter); ok && ad.deps.PickImageFile == nil && ad.deps.OpenFileDialog != nil {
					return ad.deps.OpenFileDialog(win, FileDialogOptions{
						Title:   title,
						Filters: []FileFilter{{Name: "JPEG / PNG", Extensions: []string{"jpg", "jpeg", "png"}}},
					})
				}
				if picked, pickErr := host.PickImageFile(win, title); pickErr != nil || picked != "" {
					if pickErr != nil {
						return nil, pickErr
					}
					return []string{picked}, nil
				}
				return nil, nil
			}, live, avatarimage.ReadPickedFile)
			if !live() {
				return
			}
			if controls.FinishImageImport(ticket, source, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	m.OnImageCropChange = func(crop avatarimage.Crop) {
		if viewCurrent() {
			controls.UpdateImageCrop(k, crop)
			s.UpdateAssistantTasks()
		}
	}
	m.OnApplyImageCrop = func() {
		if viewCurrent() {
			controls.ApplyImageCrop(k)
			s.UpdateAssistantTasks()
		}
	}
	m.OnCancelImageCrop = func() {
		if viewCurrent() {
			controls.CancelImageImport(k)
			s.UpdateAssistantTasks()
		}
	}
}

func (s *Service) bindBotCapabilityModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Capabilities
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	host := s.safeHost()
	rpc := host.PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		client := s.routineClient(ep, k.BotID)
		if client == nil || !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, fmt.Errorf("Bot identity changed")
		}
		writable, authErr := checkWriteAuthority(ctx, client, live, "Bot identity changed")
		if authErr != nil && !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, authErr
		}
		value, err := rpc.DshbotCapabilities(ctx, k.BotID)
		return value, authErr == nil && writable, err
	}
	m.Capabilities = assistantui.BotCapabilityModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				value, writable, err := read(ctx, live)
				if current() && controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(group, field, value string) {
			if viewCurrent() {
				controls.Change(k, group, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
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
					out, err = rpc.DshbotSaveCapabilities(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("view changed before capability save")
				}
				if current() && controls.FinishSave(ticket, out, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

func (s *Service) bindBotMemoryModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Memory
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	// Check the existing Host settings authority. Unlike catalog mutations,
	// upstream memory/replace checks only the independent track CAS revision.
	readIdentity := func(ctx context.Context, stillCurrent func() bool) (pluginrpc.DshbotCatalogItem, bool, error) {
		var bot pluginrpc.DshbotCatalogItem
		client := s.routineClient(ep, k.BotID)
		if client == nil || !stillCurrent() {
			return bot, false, fmt.Errorf("Bot identity changed")
		}
		writable, authErr := checkWriteAuthority(ctx, client, stillCurrent, "Bot identity changed")
		if authErr != nil && !stillCurrent() {
			return bot, false, authErr
		}
		view, err := rpc.DshbotRoutines(ctx, k.BotID)
		if err != nil {
			return bot, false, err
		}
		bot, ok := view.Bot(k.BotID)
		if !ok {
			return bot, false, fmt.Errorf("individual Bot unavailable")
		}
		return bot, authErr == nil && writable, nil
	}
	m.Memory = assistantui.BotMemoryModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				bot, writable, err := readIdentity(ctx, live)
				var value pluginrpc.DshbotMemoryResult
				if err == nil && live() {
					value, err = rpc.DshbotMemory(ctx, k.BotID)
				} else if err == nil {
					err = fmt.Errorf("Bot identity changed before memory read")
				}
				if current() && controls.FinishLoad(ticket, bot, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnTrack: func(track string) {
			if viewCurrent() {
				controls.SetTrack(k, track)
				s.UpdateAssistantTasks()
			}
		},
		OnEdit: func(index int) {
			if viewCurrent() {
				controls.StartEdit(k, index)
				s.UpdateAssistantTasks()
			}
		},
		OnDelete: func(index int) {
			if viewCurrent() {
				controls.PrepareDelete(k, index)
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(text string) {
			if viewCurrent() {
				controls.Change(k, text)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, command, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
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
					receipt, err = rpc.DshbotReplaceMemory(ctx, command)
				} else if err == nil {
					err = fmt.Errorf("Bot identity changed before memory write")
				}
				if current() && controls.FinishSave(ticket, receipt, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

func (s *Service) bindBotMemoryReviewModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.MemoryReview
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		client := s.routineClient(ep, k.BotID)
		if client == nil || !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, fmt.Errorf("Bot identity changed")
		}
		writable, authErr := checkWriteAuthority(ctx, client, live, "Bot identity changed")
		if authErr != nil && !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, authErr
		}
		value, err := rpc.DshbotCapabilities(ctx, k.BotID)
		return value, authErr == nil && writable, err
	}
	m.MemoryReview = assistantui.BotMemoryReviewModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				value, writable, err := read(ctx, live)
				if current() && controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(enabled bool) {
			if viewCurrent() {
				controls.SetEnabled(k, enabled)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
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
					out, err = rpc.DshbotSaveMemoryReview(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("view changed before memory review configuration save")
				}
				if current() && controls.FinishSave(ticket, out, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

func (s *Service) bindBotNotifyModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.Notify
	client := s.routineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.roomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		return readBotCapabilityStatus(ctx, client, rpc, k.BotID, live)
	}
	m.Notify = assistantui.BotNotifyModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(v bool) {
			if viewCurrent() {
				controls.SetDraft(k, v)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				if err == nil && (!writable || !pluginrpc.SameDshbotCapabilities(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotTextReviewChanged))
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				receipt, saveErr := rpc.DshbotSaveNotify(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.UpdateAssistantTasks()
			}()
		},
	}
}

func (s *Service) bindBotObjectiveModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.Objective
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	m.Objective = assistantui.BotObjectiveModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				var status pluginrpc.DshbotObjectiveStatus
				var err error
				if current() && controls.Current(ticket) {
					status, err = rpc.DshbotObjective(ctx, k.BotID)
				} else {
					err = fmt.Errorf("objective view changed before read")
				}
				if current() && controls.FinishLoad(ticket, status, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnReview: func(action string) {
			if viewCurrent() {
				controls.Prepare(k, action)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginAction(k, snapshot.Review)
			s.UpdateAssistantTasks()
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
					fresh, err = rpc.DshbotObjective(ctx, k.BotID)
					if err == nil && !pluginrpc.DshbotObjectivePreflightMatches(cmd, fresh) {
						err = fmt.Errorf("objective changed after review; no action sent")
					}
				} else {
					err = fmt.Errorf("objective view changed before preflight")
				}
				// Final dispatch fence after the read. This cannot add an atomic CAS that
				// the original start/pause endpoints do not implement; the review says so.
				if err == nil && live() {
					out, err = rpc.DshbotControlObjective(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("objective view changed before dispatch")
				}
				if current() && controls.FinishAction(ticket, out, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

func (s *Service) bindBotObjectiveConfigModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, current func() bool) {
	controls := &s.assistantRoutines.ObjectiveConfig
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		client := s.routineClient(ep, k.BotID)
		if client == nil || !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, fmt.Errorf("Bot identity changed")
		}
		writable, authErr := checkWriteAuthority(ctx, client, live, "Bot identity changed")
		if authErr != nil && !live() {
			return pluginrpc.DshbotCapabilityStatus{}, false, authErr
		}
		value, err := rpc.DshbotCapabilities(ctx, k.BotID)
		return value, authErr == nil && writable, err
	}
	m.ObjectiveConfig = assistantui.BotObjectiveConfigModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				value, writable, err := read(ctx, live)
				if current() && controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(draft pluginrpc.DshbotObjectiveConfigDraft) {
			if viewCurrent() {
				controls.SetDraft(k, draft)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
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
					out, err = rpc.DshbotSaveObjectiveConfig(ctx, cmd)
				} else if err == nil {
					err = fmt.Errorf("view changed before objective configuration save")
				}
				if current() && controls.FinishSave(ticket, out, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

func (s *Service) bindBotRouteModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.Route
	client := s.routineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.roomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotRouteStatus, bool, error) {
		var out pluginrpc.DshbotRouteStatus
		out.Options.WorkspaceLocked = true
		writable, authErr := checkWriteAuthority(ctx, client, live, "Bot view changed")
		if authErr != nil && !live() {
			return out, false, authErr
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
		return out, authErr == nil && writable, nil
	}
	m.Route = assistantui.BotRouteModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(v pluginrpc.DshbotRouteValues) {
			if viewCurrent() {
				controls.SetDraft(k, v)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				if err == nil && (!writable || !pluginrpc.SameDshbotRoute(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotRouteReviewChanged))
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				receipt, saveErr := rpc.DshbotSaveRoute(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveBotRoute retires Bot sub-editors whenever the active room/Bot client leaves.
func (s *Service) ObserveBotRoute() {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if s.roomClient(ep) == nil {
		s.assistantRoutines.TextProfile.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.SenderPolicy.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Notify.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Avatar.Bind(dshbotstore.RoutineScope{})
		s.assistantRoutines.Route.Bind(dshbotstore.RoutineScope{})
	}
}

func (s *Service) bindBotSenderPolicyModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.SenderPolicy
	client := s.routineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.roomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotSenderPolicyStatus, bool, error) {
		var out pluginrpc.DshbotSenderPolicyStatus
		writable, authErr := checkWriteAuthority(ctx, client, live, "Bot view changed")
		if authErr != nil && !live() {
			return out, false, authErr
		}
		var err error
		out, err = rpc.DshbotSenderPolicy(ctx, k.BotID)
		if err != nil {
			return out, false, err
		}
		if !live() {
			return out, false, fmt.Errorf("Bot view changed")
		}
		return out, authErr == nil && writable, nil
	}
	m.SenderPolicy = assistantui.BotSenderPolicyModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(v pluginrpc.DshbotSenderPolicyValues) {
			if viewCurrent() {
				controls.SetDraft(k, v)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnExpansionAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.AcknowledgeExpansion(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				if err == nil && (!writable || !pluginrpc.SameDshbotSenderPolicy(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotSenderReviewChanged))
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				receipt, saveErr := rpc.DshbotSaveSenderPolicy(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.UpdateAssistantTasks()
			}()
		},
	}
}

func (s *Service) bindBotTextProfileModel(m *assistantui.RoutineModel, ep harness.Endpoint, k dshbotstore.RoutineScope, parentCurrent func() bool) {
	controls := &s.assistantRoutines.TextProfile
	client := s.routineClient(ep, k.BotID)
	current := func() bool { return client != nil && parentCurrent() && s.roomClient(ep) == client }
	if !current() {
		controls.Bind(dshbotstore.RoutineScope{})
		return
	}
	controls.Bind(k)
	snapshot := controls.Snapshot(k).Value
	viewCurrent := func() bool { return current() && controls.ViewCurrent(k, snapshot.Generation) }
	rpc := s.safeHost().PluginRPCClient(ep)
	read := func(ctx context.Context, live func() bool) (pluginrpc.DshbotCapabilityStatus, bool, error) {
		return readBotCapabilityStatus(ctx, client, rpc, k.BotID, live)
	}
	m.TextProfile = assistantui.BotTextProfileModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !viewCurrent() {
				return
			}
			ticket, ok := controls.BeginLoad(k)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.LoadCurrent(ticket) }
				value, writable, err := read(ctx, live)
				if controls.FinishLoad(ticket, value, writable, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnChange: func(v pluginrpc.DshbotTextProfileValues) {
			if viewCurrent() {
				controls.SetDraft(k, v)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if viewCurrent() {
				controls.Discard(k)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if viewCurrent() {
				controls.Prepare(k)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if viewCurrent() {
				controls.Dismiss(k, snapshot.Review)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if viewCurrent() {
				controls.Acknowledge(k, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func() {
			if viewCurrent() {
				controls.Adopt(k)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !viewCurrent() {
				return
			}
			ticket, cmd, ok := controls.BeginSave(k, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				live := func() bool { return current() && controls.Current(ticket) }
				fresh, writable, err := read(ctx, live)
				if err == nil && (!writable || !pluginrpc.SameDshbotCapabilities(cmd.Expected, fresh) || fresh.WriteError != "") {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotTextReviewChanged))
				}
				if err != nil || !live() || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				receipt, saveErr := rpc.DshbotSaveTextProfile(ctx, cmd)
				controls.FinishSave(ticket, receipt, saveErr)
				s.UpdateAssistantTasks()
			}()
		},
	}
}

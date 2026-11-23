package assistantctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

// BotCreateModel builds the personal Bot creator model for the active Host/account.
func (s *Service) BotCreateModel(ep harness.Endpoint) assistantui.BotCreateModel {
	if s == nil {
		return assistantui.BotCreateModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	originalAssistant := s.AssistantStore()
	scope, binding := "", ""
	if client != nil {
		scope = endpointScope(ep.Origin, ep.Cookie)
		binding = clientBinding(client)
	}
	rpc := host.PluginRPCClient(ep)
	controls := &s.assistantBotCreator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	modelSessions := host.Sessions()
	var modelOptions []session.ModelOption
	var workspaceOptions []session.WorkspaceRow
	modelsKnown := false
	workspacesKnown := false
	if modelSessions != nil {
		modelOptions, modelsKnown = modelSessions.CachedModelCatalog(client)
		workspaceOptions, workspacesKnown = modelSessions.CachedWorkspaceChoices(client)
	}
	current := func() bool {
		sameAssistant := originalAssistant != nil && s.AssistantStore() == originalAssistant
		return sameAssistant && client != nil && s.roomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	read := func(ctx context.Context) (pluginrpc.DshbotBotCatalog, bool, error) {
		writable, authErr := checkWriteAuthority(ctx, client, current, "Host or account changed")
		if authErr != nil && !current() {
			return pluginrpc.DshbotBotCatalog{}, false, authErr
		}
		catalog, err := rpc.DshbotBotCatalog(ctx)
		return catalog, authErr == nil && writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, catalog, writable, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	captureOpen := func() (*session.Store, *session.BotCreationOpenTarget) {
		sessions := host.Sessions()
		if sessions == nil {
			return nil, nil
		}
		return sessions, sessions.CaptureBotCreationOpen(client)
	}
	openCreated := func(item pluginrpc.DshbotBotItem, cwd string, sessions *session.Store, target *session.BotCreationOpenTarget) {
		if !current() || item.ID == "" {
			return
		}
		if target == nil {
			s.mu.Lock()
			s.pluginDetail = i18n.T(i18n.KeyCmdBotCreatedSessionConnectionUnavailab)
			s.lastAct = s.pluginDetail
			s.mu.Unlock()
			s.UpdateAssistantTasks()
			return
		}
		if !target.Current() {
			return
		}
		s.openCreatedBot(ep, client, current, item, cwd, sessions, target)
	}
	return assistantui.BotCreateModel{
		State:           snapshot,
		HostLabel:       ep.Origin,
		ModelOptions:    modelOptions,
		ModelsKnown:     modelsKnown,
		Workspaces:      workspaceOptions,
		WorkspacesKnown: workspacesKnown,
		OnRefresh:       refresh,
		OnSelectModel: func(v pluginrpc.DshbotRouteModel) {
			if current() {
				controls.ChangeModel(scope, v)
				s.UpdateAssistantTasks()
			}
		},
		OnStart: func() {
			if current() {
				sessions := host.Sessions()
				cwd := ""
				if sessions != nil {
					cwd = sessions.ScratchCwd()
				}
				controls.Start(scope, cwd)
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				controls.Discard(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				controls.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func(id string) {
			if current() && controls.Adopt(scope, id) {
				s.UpdateAssistantTasks()
			}
		},
		OnOpen: func() {
			sessions, target := captureOpen()
			openCreated(snapshot.Created, snapshot.CreatedScratchCwd, sessions, target)
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			openSessions, openTarget := captureOpen()
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotCatalog(cmd.Expected, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotCatalogReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// objective start, model generation, auto-retry or revision substitution.
				result, createErr := rpc.DshbotCreateBot(ctx, cmd)
				controls.Finish(ticket, result, createErr)
				if createErr == nil {
					host.RunOnMain(func() {
						if !current() {
							return
						}
						items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
						for i, row := range result.Catalog.Items {
							items[i] = row.DshbotCatalogItem
						}
						s.updateRosterItems(ep.Origin, result.Catalog.Revision, items)
						openCreated(result.Item, cmd.Draft.ScratchCwd, openSessions, openTarget)
					})
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveBotCreator suspends the Bot creator when unrendered or offline.
func (s *Service) ObserveBotCreator(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantBotCreator.Suspend()
	}
}

func (s *Service) openCreatedBot(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotBotItem, cwd string, originalSessions *session.Store, target *session.BotCreationOpenTarget) {
	if client == nil || current == nil || !current() || target == nil || !target.Current() || !s.DshbotCurrent(ep, client) {
		return
	}
	host := s.safeHost()
	asst := s.AssistantStore()
	sessions := host.Sessions()
	win := host.Window()
	if asst == nil || sessions == nil || sessions != originalSessions || win == nil {
		return
	}
	label := i18n.T(i18n.KeyCmdCreatedBotOpenCanonicalSession)
	s.mu.Lock()
	if s.pluginBusy {
		s.mu.Unlock()
		return
	}
	s.pluginBusy = true
	s.pluginDetail = label + "…"
	s.lastAct = s.pluginDetail
	s.mu.Unlock()

	asst.SelectDshbot(expected.ID, ep.Origin)
	live := func() bool {
		if !current() || !s.DshbotCurrent(ep, client) {
			return false
		}
		same := s.AssistantStore() == asst && host.Sessions() == sessions && host.Window() == win
		selected := asst.Snapshot().Value
		return same && selected.DshbotID == expected.ID && selected.DshbotOrigin == ep.Origin && target.Current()
	}
	s.UpdateAssistantTasks()
	rpc := host.PluginRPCClient(ep)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		open := func() (string, error) {
			if !live() {
				return "", fmt.Errorf("created Bot opening expired before catalog read")
			}
			catalog, err := rpc.DshbotBotCatalog(ctx)
			if err != nil {
				return "", err
			}
			if !live() {
				return "", fmt.Errorf("created Bot opening expired after catalog read")
			}
			var item pluginrpc.DshbotCatalogItem
			for _, row := range catalog.Items {
				if row.ID == expected.ID {
					item = row.DshbotCatalogItem
					break
				}
			}
			if item.ID != expected.ID || item.Kind != "bot" || item.Hidden || item.SessionID != expected.SessionID || item.WorkspaceID != expected.WorkspaceID {
				return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdCreatedBotIdentityChanged))
			}
			if !live() {
				return "", fmt.Errorf("created Bot opening expired before canonical open")
			}
			sid, err := rpc.DshbotOpen(ctx, item, catalog.Revision, cwd)
			if err != nil {
				return "", fmt.Errorf(i18n.T(i18n.KeyCmdCreatedBotOpenFailed), err)
			}
			if !live() {
				return "", fmt.Errorf("created Bot opening expired after canonical receipt")
			}
			return sid, nil
		}
		sid, err := open()
		host.RunOnMain(func() {
			valid := live()
			s.mu.Lock()
			s.pluginBusy = false
			if valid {
				if err != nil {
					s.pluginDetail = label + ": " + err.Error()
				} else {
					s.pluginDetail = label + ": " + sid
				}
				s.lastAct = s.pluginDetail
			}
			s.mu.Unlock()
			if valid && err == nil && live() {
				if target.Select(sid) {
					asst.SetSessionID(sid)
					host.RevokePhoneGit()
					s.assistantBotCreator.Suspend()
					s.assistantRoomCreator.Suspend()
					s.assistantRoomEditor.Suspend()
					s.assistantRoomRuntime.Suspend()
					s.SetOpen(false)
					host.RevealMainConversation()
				} else {
					s.mu.Lock()
					s.pluginDetail = label + i18n.T(i18n.KeyCmdBotOpenBindingExpired)
					s.lastAct = s.pluginDetail
					s.mu.Unlock()
				}
			}
			s.UpdateAssistantTasks()
		})
	}()
}

// BotDeleteModel builds the Bot/room deletion review model for sourceID.
func (s *Service) BotDeleteModel(ep harness.Endpoint, sourceID string) assistantui.BotDeleteModel {
	if s == nil {
		return assistantui.BotDeleteModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	originalAssistant := s.AssistantStore()
	scope, binding := "", ""
	if client != nil && sourceID != "" {
		scope = endpointScope(ep.Origin, ep.Cookie, sourceID)
		binding = clientBinding(client)
	}
	rpc := host.PluginRPCClient(ep)
	controls := &s.assistantBotDeleter
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	current := func() bool {
		same := originalAssistant != nil && s.AssistantStore() == originalAssistant
		if !same || client == nil || s.roomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		selected := originalAssistant.Snapshot().Value
		return selected.DshbotID == sourceID && selected.DshbotOrigin == ep.Origin
	}
	authority := func(ctx context.Context) (bool, error) {
		return checkWriteAuthority(ctx, client, current, "original Host/account/view expired")
	}
	readPlan := func(ctx context.Context) (pluginrpc.DshbotBotDeletePlan, bool, error) {
		writable, err := authority(ctx)
		if err != nil {
			return pluginrpc.DshbotBotDeletePlan{}, false, err
		}
		p, err := rpc.DshbotBotDeletePlan(ctx, sourceID)
		return p, writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var p pluginrpc.DshbotBotDeletePlan
			var catalog pluginrpc.DshbotDeleteCatalog
			var writable bool
			var err error
			if snapshot.Uncertain && snapshot.Pending != nil { // Observation only: never another preview/command for an absent target.
				writable, err = authority(ctx)
				if err == nil {
					catalog, err = rpc.DshbotBotDeletionCatalog(ctx)
				}
			} else {
				p, writable, err = readPlan(ctx)
			}
			if current() && controls.FinishLoad(ticket, p, catalog, writable, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	return assistantui.BotDeleteModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		SourceID:  sourceID,
		OnRefresh: refresh,
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnCancel: func() {
			if current() {
				controls.Cancel(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnIdentity: func(id, name string) {
			if current() {
				controls.ConfirmIdentity(scope, snapshot.Review, id, name)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := readPlan(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotDeletePlan(cmd.Plan, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotDeleteReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				// Original ID/revision only, one mutation. No separate stop, file removal,
				// retry, session opening or automatic recovery. The Host owns its cascade.
				result, deleteErr := rpc.DshbotDeleteBot(ctx, cmd)
				controls.Finish(ticket, result, deleteErr)
				if deleteErr == nil {
					host.RunOnMain(func() {
						if !current() {
							return
						}
						items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
						for i, row := range result.Catalog.Items {
							items[i] = row.DshbotCatalogItem
						}
						s.updateRosterItems(ep.Origin, result.Catalog.Revision, items)
					})
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveBotDeleter suspends the Bot deleter when unrendered or offline.
func (s *Service) ObserveBotDeleter(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantBotDeleter.Suspend()
	}
}

// BotDuplicateModel builds the Bot duplication review model for sourceID.
func (s *Service) BotDuplicateModel(ep harness.Endpoint, sourceID string) assistantui.BotDuplicateModel {
	if s == nil {
		return assistantui.BotDuplicateModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	originalAssistant := s.AssistantStore()
	scope, binding := "", ""
	if client != nil && sourceID != "" {
		scope = endpointScope(ep.Origin, ep.Cookie, sourceID)
		binding = clientBinding(client)
	}
	rpc := host.PluginRPCClient(ep)
	controls := &s.assistantBotDuplicator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	current := func() bool {
		sameAssistant := originalAssistant != nil && s.AssistantStore() == originalAssistant
		if !sameAssistant || client == nil || s.roomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		selected := originalAssistant.Snapshot().Value
		return selected.DshbotID == sourceID && selected.DshbotOrigin == ep.Origin
	}
	read := func(ctx context.Context) (pluginrpc.DshbotBotCatalog, bool, error) {
		writable, authErr := checkWriteAuthority(ctx, client, current, "Host or account changed")
		if authErr != nil && !current() {
			return pluginrpc.DshbotBotCatalog{}, false, authErr
		}
		catalog, err := rpc.DshbotBotCatalog(ctx)
		return catalog, authErr == nil && writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, catalog, writable, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	return assistantui.BotDuplicateModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		SourceID:  sourceID,
		OnRefresh: refresh,
		OnStart: func() {
			if current() {
				sessions := host.Sessions()
				cwd := ""
				if sessions != nil {
					cwd = sessions.ScratchCwd()
				}
				controls.Start(scope, sourceID, cwd)
				s.UpdateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.UpdateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				controls.Discard(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				controls.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledgeExecution: func(ack bool) {
			if current() {
				controls.AcknowledgeExecution(scope, snapshot.Review, ack)
				s.UpdateAssistantTasks()
			}
		},
		OnOpen: func(id string) {
			if !current() {
				return
			}
			var item pluginrpc.DshbotBotItem
			cwd := snapshot.CreatedScratchCwd
			if snapshot.Created.ID == id {
				item = snapshot.Created
			} else if snapshot.Uncertain && snapshot.Pending != nil {
				for _, candidate := range snapshot.Candidates {
					if candidate.ID == id {
						item = candidate
						cwd = snapshot.Pending.ScratchCwd
						break
					}
				}
			}
			if item.ID == "" {
				return
			}
			sessions := host.Sessions()
			if sessions == nil {
				return
			}
			target := sessions.CaptureBotCreationOpen(client)
			s.openDuplicateBot(ep, client, current, item, cwd, sessions, target)
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotCatalog(cmd.Expected, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotCatalogReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// extra prompt/goal RPC, automatic opening, retry or revision substitution.
				result, createErr := rpc.DshbotDuplicateBot(ctx, cmd)
				controls.Finish(ticket, result, createErr)
				if createErr == nil {
					host.RunOnMain(func() {
						if !current() {
							return
						}
						items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
						for i, row := range result.Catalog.Items {
							items[i] = row.DshbotCatalogItem
						}
						s.updateRosterItems(ep.Origin, result.Catalog.Revision, items)
					})
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveBotDuplicator suspends the Bot duplicator when unrendered or offline.
func (s *Service) ObserveBotDuplicator(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantBotDuplicator.Suspend()
	}
}

func (s *Service) openDuplicateBot(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotBotItem, cwd string, originalSessions *session.Store, target *session.BotCreationOpenTarget) {
	if client == nil || current == nil || !current() || !s.DshbotCurrent(ep, client) {
		return
	}
	if target == nil {
		s.mu.Lock()
		s.pluginDetail = i18n.T(i18n.KeyCmdCatalogRecordRetainedSessionConnecti)
		s.lastAct = s.pluginDetail
		s.mu.Unlock()
		s.UpdateAssistantTasks()
		return
	}
	if !target.Current() {
		return
	}
	host := s.safeHost()
	asst := s.AssistantStore()
	sessions := host.Sessions()
	win := host.Window()
	if asst == nil || sessions == nil || sessions != originalSessions || win == nil {
		return
	}
	label := i18n.T(i18n.KeyCmdOpenInspectedCatalogBot)
	s.mu.Lock()
	if s.pluginBusy {
		s.mu.Unlock()
		return
	}
	s.pluginBusy = true
	s.pluginDetail = label + "…"
	s.lastAct = s.pluginDetail
	s.mu.Unlock()

	live := func() bool {
		if !current() || !s.DshbotCurrent(ep, client) {
			return false
		}
		same := s.AssistantStore() == asst && host.Sessions() == sessions && host.Window() == win
		return same && target.Current()
	}
	s.UpdateAssistantTasks()
	rpc := host.PluginRPCClient(ep)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		open := func() (string, error) {
			if !live() {
				return "", fmt.Errorf("catalog Bot opening expired before catalog read")
			}
			catalog, err := rpc.DshbotBotCatalog(ctx)
			if err != nil {
				return "", err
			}
			if !live() {
				return "", fmt.Errorf("catalog Bot opening expired after catalog read")
			}
			var item pluginrpc.DshbotBotItem
			for _, row := range catalog.Items {
				if row.ID == expected.ID {
					item = row
					break
				}
			}
			if !pluginrpc.SameDshbotBotDuplicateOpenTarget(expected, item) {
				return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotDuplicateIdentityChanged))
			}
			if !live() {
				return "", fmt.Errorf("catalog Bot opening expired before canonical open")
			}
			sid, err := rpc.DshbotOpen(ctx, item.DshbotCatalogItem, catalog.Revision, cwd)
			if err != nil {
				return "", fmt.Errorf(i18n.T(i18n.KeyCmdDuplicatedBotOpenFailed), err)
			}
			if !live() {
				return "", fmt.Errorf("catalog Bot opening expired after canonical receipt")
			}
			return sid, nil
		}
		sid, err := open()
		host.RunOnMain(func() {
			valid := live()
			s.mu.Lock()
			s.pluginBusy = false
			if valid {
				if err != nil {
					s.pluginDetail = label + ": " + err.Error()
				} else {
					s.pluginDetail = label + ": " + sid
				}
				s.lastAct = s.pluginDetail
			}
			s.mu.Unlock()
			if valid && err == nil && live() {
				if target.Select(sid) {
					asst.SelectDshbot(expected.ID, ep.Origin)
					asst.SetSessionID(sid)
					host.RevokePhoneGit()
					s.assistantBotDuplicator.Suspend()
					s.assistantBotCreator.Suspend()
					s.assistantRoomCreator.Suspend()
					s.assistantRoomEditor.Suspend()
					s.assistantRoomRuntime.Suspend()
					s.SetOpen(false)
					host.RevealMainConversation()
				} else {
					s.mu.Lock()
					s.pluginDetail = label + i18n.T(i18n.KeyCmdBotOpenBindingExpired)
					s.lastAct = s.pluginDetail
					s.mu.Unlock()
				}
			}
			s.UpdateAssistantTasks()
		})
	}()
}

// RevealBotVisibility reveals a hidden/unrepresented notification target inside the visibility manager.
func (s *Service) RevealBotVisibility(ep harness.Endpoint, client *dsh.Client, botID, sessionID string) bool {
	if s == nil || client == nil || s.roomClient(ep) != client || botID == "" || sessionID == "" {
		return false
	}
	scope := endpointScope(ep.Origin, ep.Cookie)
	s.assistantBotVisibility.Bind(scope, clientBinding(client))
	ok := s.assistantBotVisibility.Reveal(scope, botID, sessionID)
	if ok {
		s.UpdateAssistantTasks()
	}
	return ok
}

// BotVisibilityModel builds the Bot/room hidden visibility review model.
func (s *Service) BotVisibilityModel(ep harness.Endpoint) assistantui.BotVisibilityModel {
	if s == nil {
		return assistantui.BotVisibilityModel{}
	}
	host := s.safeHost()
	client := s.roomClient(ep)
	originalAssistant := s.AssistantStore()
	scope, binding := "", ""
	if client != nil {
		scope = endpointScope(ep.Origin, ep.Cookie)
		binding = clientBinding(client)
	}
	controls := &s.assistantBotVisibility
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	rpc := host.PluginRPCClient(ep)
	current := func() bool {
		same := originalAssistant != nil && s.AssistantStore() == originalAssistant
		return same && client != nil && s.roomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	read := func(ctx context.Context) (pluginrpc.DshbotBotCatalog, bool, error) {
		writable, authErr := checkWriteAuthority(ctx, client, current, "original Host/account/view expired")
		if authErr != nil && !current() {
			return pluginrpc.DshbotBotCatalog{}, false, authErr
		}
		catalog, err := rpc.DshbotBotCatalog(ctx)
		return catalog, authErr == nil && writable, err
	}
	updateRoster := func(catalog pluginrpc.DshbotBotCatalog) {
		host.RunOnMain(func() {
			if !current() {
				return
			}
			items := make([]pluginrpc.DshbotCatalogItem, len(catalog.Items))
			for i, v := range catalog.Items {
				items[i] = v.DshbotCatalogItem
			}
			s.updateRosterItems(ep.Origin, catalog.Revision, items)
		})
	}
	return assistantui.BotVisibilityModel{
		State:     snapshot,
		HostLabel: ep.Origin,
		OnRefresh: func() {
			if !current() {
				return
			}
			ticket, ok := controls.BeginLoad(scope)
			if !ok {
				return
			}
			s.UpdateAssistantTasks()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				catalog, writable, err := read(ctx)
				if current() && controls.FinishLoad(ticket, catalog, writable, err) {
					if err == nil {
						updateRoster(catalog)
					}
					s.UpdateAssistantTasks()
				}
			}()
		},
		OnShowHidden: func(on bool) {
			if current() {
				controls.Show(scope, on)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func(id, sessionID string) {
			if current() {
				controls.Prepare(scope, id, sessionID, time.Now().UnixMilli())
				s.UpdateAssistantTasks()
			}
		},
		OnCancel: func() {
			if current() {
				controls.Cancel(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnAcknowledge: func(on bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, on)
				s.UpdateAssistantTasks()
			}
		},
		OnAdopt: func(id, sessionID string) {
			if current() && controls.Adopt(scope, id, sessionID) {
				updateRoster(controls.Snapshot(scope).Catalog)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, cmd, ok := controls.Begin(scope, snapshot.Review)
			s.UpdateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotCatalog(cmd.Expected, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotVisibilityReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.UpdateAssistantTasks()
					return
				}
				result, saveErr := rpc.DshbotSetVisibility(ctx, cmd)
				controls.Finish(ticket, result, saveErr)
				if saveErr == nil {
					updateRoster(result.Catalog)
				}
				s.UpdateAssistantTasks()
			}()
		},
	}
}

// ObserveBotVisibility suspends visibility management when unrendered or offline.
func (s *Service) ObserveBotVisibility(rendered bool) {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	if !rendered || s.roomClient(ep) == nil {
		s.assistantBotVisibility.Suspend()
	}
}

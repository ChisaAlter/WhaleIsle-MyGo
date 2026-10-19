package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantBotCreateModel(ep harness.Endpoint) assistantui.BotCreateModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	originalAssistant := s.assistant
	s.mu.Unlock()
	scope, binding := "", ""
	if client != nil {
		digest := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie))
		scope = fmt.Sprintf("%x", digest)
		binding = fmt.Sprintf("%p", client)
	}
	rpc := s.pluginRPCClient(ep)
	controls := &s.assistantBotCreator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	s.mu.Lock()
	modelSessions := s.sessions
	s.mu.Unlock()
	var modelOptions []session.ModelOption
	var workspaceOptions []session.WorkspaceRow
	modelsKnown := false
	workspacesKnown := false
	if modelSessions != nil {
		modelOptions, modelsKnown = modelSessions.CachedModelCatalog(client)
		workspaceOptions, workspacesKnown = modelSessions.CachedWorkspaceChoices(client)
	}
	current := func() bool {
		s.mu.Lock()
		sameAssistant := originalAssistant != nil && s.assistant == originalAssistant
		s.mu.Unlock()
		return sameAssistant && client != nil && s.assistantRoomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	read := func(ctx context.Context) (pluginrpc.DshbotBotCatalog, bool, error) {
		var authority struct {
			Writable bool `json:"writable"`
		}
		if !current() {
			return pluginrpc.DshbotBotCatalog{}, false, fmt.Errorf("Host or account changed")
		}
		authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !current() {
			return pluginrpc.DshbotBotCatalog{}, false, fmt.Errorf("Host or account changed")
		}
		catalog, err := rpc.DshbotBotCatalog(ctx)
		return catalog, authorityErr == nil && authority.Writable, err
	}
	refresh := func() {
		if !current() {
			return
		}
		ticket, ok := controls.BeginLoad(scope)
		if !ok {
			return
		}
		s.updateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			catalog, writable, err := read(ctx)
			if current() && controls.FinishLoad(ticket, catalog, writable, err) {
				s.updateAssistantTasks()
			}
		}()
	}
	captureOpen := func() (*session.Store, *session.BotCreationOpenTarget) {
		s.mu.Lock()
		sessions := s.sessions
		s.mu.Unlock()
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
			s.assistantPluginDetail = "Bot 已创建；当前会话连接暂不可用于打开。保留已有身份，连接恢复后单独打开 / Bot created; session connection unavailable, open the existing Bot after reconnecting"
			s.assistantLastAct = s.assistantPluginDetail
			s.mu.Unlock()
			s.updateAssistantTasks()
			return
		}
		if !target.Current() {
			return
		}
		s.openAssistantCreatedBot(ep, client, current, item, cwd, sessions, target)
	}
	return assistantui.BotCreateModel{State: snapshot, HostLabel: ep.Origin, ModelOptions: modelOptions, ModelsKnown: modelsKnown, Workspaces: workspaceOptions, WorkspacesKnown: workspacesKnown, OnRefresh: refresh,
		OnSelectModel: func(v pluginrpc.DshbotRouteModel) {
			if current() {
				controls.ChangeModel(scope, v)
				s.updateAssistantTasks()
			}
		},
		OnStart: func() {
			if current() {
				s.mu.Lock()
				sessions := s.sessions
				s.mu.Unlock()
				cwd := ""
				if sessions != nil {
					cwd = sessions.Snapshot().ScratchCwd
				}
				controls.Start(scope, cwd)
				s.updateAssistantTasks()
			}
		},
		OnChange: func(field, value string) {
			if current() {
				controls.Change(scope, field, value)
				s.updateAssistantTasks()
			}
		},
		OnDiscard: func() {
			if current() {
				controls.Discard(scope)
				s.updateAssistantTasks()
			}
		},
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				controls.Dismiss(scope)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
				s.updateAssistantTasks()
			}
		},
		OnAdopt: func(id string) {
			if current() && controls.Adopt(scope, id) {
				s.updateAssistantTasks()
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
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				fresh, writable, err := read(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotCatalog(cmd.Expected, fresh)) {
					err = fmt.Errorf("catalog identities/revision or write authority changed; refresh and review again")
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// objective start, model generation, auto-retry or revision substitution.
				result, createErr := rpc.DshbotCreateBot(ctx, cmd)
				controls.Finish(ticket, result, createErr)
				if createErr == nil {
					mygo.RunOnMain(func() {
						if !current() {
							return
						}
						s.mu.Lock()
						roster := s.dshbotRoster
						s.mu.Unlock()
						if roster != nil {
							items := make([]pluginrpc.DshbotCatalogItem, len(result.Catalog.Items))
							for i, row := range result.Catalog.Items {
								items[i] = row.DshbotCatalogItem
							}
							roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Revision: result.Catalog.Revision, Value: pluginrpc.DshbotCatalogValue{Items: items}}}, false, ep.Origin)
						}
						openCreated(result.Item, cmd.Draft.ScratchCwd, openSessions, openTarget)
					})
				}
				s.updateAssistantTasks()
			}()
		},
	}
}

// Observe the latest state after rendering, including navigation performed by
// an earlier callback in this frame. Suspends retire old view callbacks, while
// a dispatched create still settles only its originating account's attempt.
func (s *mainState) observeAssistantBotCreator(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantBotCreator.Suspend()
	}
}

// Opening is separate from creation and is bound to the confirmed client and
// view generation. No replacement client or newly opened view can adopt it.
// A failed/stale open never resends bot/create or discards the created identity.
func (s *mainState) openAssistantCreatedBot(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotBotItem, cwd string, originalSessions *session.Store, target *session.BotCreationOpenTarget) {
	if client == nil || current == nil || !current() || target == nil || !target.Current() || !s.assistantDshbotCurrent(ep, client) {
		return
	}
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	if asst == nil || sessions == nil || sessions != originalSessions || win == nil || s.assistantPluginBusy {
		s.mu.Unlock()
		return
	}
	s.assistantPluginBusy = true
	label := "已创建 Bot：打开专属会话 / Created Bot: open canonical session"
	s.assistantPluginDetail = label + "…"
	s.assistantLastAct = s.assistantPluginDetail
	s.mu.Unlock()
	asst.SelectDshbot(expected.ID, ep.Origin)
	live := func() bool {
		if !current() || !s.assistantDshbotCurrent(ep, client) {
			return false
		}
		s.mu.Lock()
		same := s.assistant == asst && s.sessions == sessions && s.win == win
		s.mu.Unlock()
		selected := asst.Snapshot()
		return same && selected.DshbotID == expected.ID && selected.DshbotOrigin == ep.Origin && target.Current()
	}
	s.updateAssistantTasks()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		rpc := s.pluginRPCClient(ep) // immutable reviewed endpoint/account, never rebound
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
				return "", fmt.Errorf("Bot already exists, but its canonical session/workspace changed; inspect it before opening")
			}
			if !live() {
				return "", fmt.Errorf("created Bot opening expired before canonical open")
			}
			sid, err := rpc.DshbotOpen(ctx, item, catalog.Revision, cwd)
			if err != nil {
				return "", fmt.Errorf("Bot already exists; canonical opening failed: %w", err)
			}
			if !live() {
				return "", fmt.Errorf("created Bot opening expired after canonical receipt")
			}
			return sid, nil
		}
		sid, err := open()
		// Deliver on the UI thread, so close/hide/mode/selection actions cannot
		// interleave between the final view check and local session navigation.
		mygo.RunOnMain(func() {
			valid := live()
			s.mu.Lock()
			s.assistantPluginBusy = false // this worker exclusively owns the busy flag
			if valid {
				if err != nil {
					s.assistantPluginDetail = label + ": " + err.Error()
				} else {
					s.assistantPluginDetail = label + ": " + sid
				}
				s.assistantLastAct = s.assistantPluginDetail
			}
			s.mu.Unlock()
			if valid && err == nil && live() {
				if target.Select(sid) {
					asst.SetSessionID(sid)
					s.revokePhoneGit()
					s.assistantBotCreator.Suspend()
					s.assistantRoomCreator.Suspend()
					s.assistantRoomEditor.Suspend()
					s.assistantRoomRuntime.Suspend()
					s.mu.Lock()
					s.assistantConversationGeneration++
					s.assistantOpen = false
					s.mu.Unlock()
				} else {
					s.mu.Lock()
					s.assistantPluginDetail = label + ": original Host/session binding expired before navigation"
					s.assistantLastAct = s.assistantPluginDetail
					s.mu.Unlock()
				}
			}
			s.updateAssistantTasks()
		})
	}()
}

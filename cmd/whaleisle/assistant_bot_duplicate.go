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
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) assistantBotDuplicateModel(ep harness.Endpoint, sourceID string) assistantui.BotDuplicateModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	originalAssistant := s.assistant
	s.mu.Unlock()
	scope, binding := "", ""
	if client != nil && sourceID != "" {
		digest := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie + "\x00" + sourceID))
		scope = fmt.Sprintf("%x", digest)
		binding = fmt.Sprintf("%p", client)
	}
	rpc := s.pluginRPCClient(ep)
	controls := &s.assistantBotDuplicator
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	current := func() bool {
		s.mu.Lock()
		sameAssistant := originalAssistant != nil && s.assistant == originalAssistant
		s.mu.Unlock()
		if !sameAssistant || client == nil || s.assistantRoomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		selected := originalAssistant.Snapshot()
		return selected.DshbotID == sourceID && selected.DshbotOrigin == ep.Origin
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
	return assistantui.BotDuplicateModel{State: snapshot, HostLabel: ep.Origin, SourceID: sourceID, OnRefresh: refresh,
		OnStart: func() {
			if current() {
				s.mu.Lock()
				sessions := s.sessions
				s.mu.Unlock()
				cwd := ""
				if sessions != nil {
					cwd = sessions.ScratchCwd()
				}
				controls.Start(scope, sourceID, cwd)
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
		OnAcknowledgeExecution: func(ack bool) {
			if current() {
				controls.AcknowledgeExecution(scope, snapshot.Review, ack)
				s.updateAssistantTasks()
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
			s.mu.Lock()
			sessions := s.sessions
			s.mu.Unlock()
			if sessions == nil {
				return
			}
			target := sessions.CaptureBotCreationOpen(client)
			s.openAssistantDuplicateBot(ep, client, current, item, cwd, sessions, target)
		},
		OnConfirm: func() {
			if !current() {
				return
			}
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
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotCatalogReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Dispatch once, using only the reviewed identities and revision. No prompt,
				// extra prompt/goal RPC, automatic opening, retry or revision substitution.
				result, createErr := rpc.DshbotDuplicateBot(ctx, cmd)
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
					})
				}
				s.updateAssistantTasks()
			}()
		},
	}
}

// Observe the latest state after rendering, including navigation performed by
// an earlier callback in this frame. Suspends retire old view callbacks, while
// a dispatched duplicate still settles only its originating account/source attempt.
func (s *mainState) observeAssistantBotDuplicator(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantBotDuplicator.Suspend()
	}
}

// Opening is separate from creation and is bound to the confirmed client and
// view generation. No replacement client or newly opened view can adopt it.
// A failed/stale open never resends bot/duplicate or settles an unknown attempt.
func (s *mainState) openAssistantDuplicateBot(ep harness.Endpoint, client *dsh.Client, current func() bool, expected pluginrpc.DshbotBotItem, cwd string, originalSessions *session.Store, target *session.BotCreationOpenTarget) {
	if client == nil || current == nil || !current() || !s.assistantDshbotCurrent(ep, client) {
		return
	}
	if target == nil {
		s.mu.Lock()
		s.assistantPluginDetail = i18n.T(i18n.KeyCmdCatalogRecordRetainedSessionConnecti)
		s.assistantLastAct = s.assistantPluginDetail
		s.mu.Unlock()
		s.updateAssistantTasks()
		return
	}
	if !target.Current() {
		return
	}
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	if asst == nil || sessions == nil || sessions != originalSessions || win == nil || s.assistantPluginBusy {
		s.mu.Unlock()
		return
	}
	s.assistantPluginBusy = true
	label := i18n.T(i18n.KeyCmdOpenInspectedCatalogBot)
	s.assistantPluginDetail = label + "…"
	s.assistantLastAct = s.assistantPluginDetail
	s.mu.Unlock()
	live := func() bool {
		if !current() || !s.assistantDshbotCurrent(ep, client) {
			return false
		}
		s.mu.Lock()
		same := s.assistant == asst && s.sessions == sessions && s.win == win
		s.mu.Unlock()
		return same && target.Current()
	}
	s.updateAssistantTasks()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		rpc := s.pluginRPCClient(ep) // immutable reviewed endpoint/account, never rebound
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
					asst.SelectDshbot(expected.ID, ep.Origin)
					asst.SetSessionID(sid)
					s.revokePhoneGit()
					s.assistantBotDuplicator.Suspend()
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
					s.assistantPluginDetail = label + i18n.T(i18n.KeyCmdBotOpenBindingExpired)
					s.assistantLastAct = s.assistantPluginDetail
					s.mu.Unlock()
				}
			}
			s.updateAssistantTasks()
		})
	}()
}

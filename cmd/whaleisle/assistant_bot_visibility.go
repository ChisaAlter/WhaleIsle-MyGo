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
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func assistantVisibilityScope(ep harness.Endpoint) string {
	digest := sha256.Sum256([]byte(ep.Origin + "\x00" + ep.Cookie))
	return fmt.Sprintf("%x", digest)
}

// Notification navigation only: never write hidden, select a missing roster
// entry, open a session or issue an RPC. Caller first activates Assistant/Bots.
func (s *mainState) revealAssistantBotVisibility(ep harness.Endpoint, client *dsh.Client, botID, sessionID string) bool {
	if client == nil || s.assistantRoomClient(ep) != client || botID == "" || sessionID == "" {
		return false
	}
	scope := assistantVisibilityScope(ep)
	s.assistantBotVisibility.Bind(scope, fmt.Sprintf("%p", client))
	ok := s.assistantBotVisibility.Reveal(scope, botID, sessionID)
	if ok {
		s.updateAssistantTasks()
	}
	return ok
}
func (s *mainState) assistantBotVisibilityModel(ep harness.Endpoint) assistantui.BotVisibilityModel {
	client := s.assistantRoomClient(ep)
	s.mu.Lock()
	originalAssistant := s.assistant
	s.mu.Unlock()
	scope, binding := "", ""
	if client != nil {
		scope = assistantVisibilityScope(ep)
		binding = fmt.Sprintf("%p", client)
	}
	controls := &s.assistantBotVisibility
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	rpc := s.pluginRPCClient(ep)
	current := func() bool {
		s.mu.Lock()
		same := originalAssistant != nil && s.assistant == originalAssistant
		s.mu.Unlock()
		return same && client != nil && s.assistantRoomClient(ep) == client && controls.ViewCurrent(scope, snapshot.Generation)
	}
	read := func(ctx context.Context) (pluginrpc.DshbotBotCatalog, bool, error) {
		var authority struct {
			Writable bool `json:"writable"`
		}
		if !current() {
			return pluginrpc.DshbotBotCatalog{}, false, fmt.Errorf("original Host/account/view expired")
		}
		authorityErr := client.Call(ctx, "settings/describe", nil, &authority)
		if !current() {
			return pluginrpc.DshbotBotCatalog{}, false, fmt.Errorf("original Host/account/view expired")
		}
		catalog, err := rpc.DshbotBotCatalog(ctx)
		return catalog, authorityErr == nil && authority.Writable, err
	}
	updateRoster := func(catalog pluginrpc.DshbotBotCatalog) {
		mygo.RunOnMain(func() {
			if !current() {
				return
			}
			s.mu.Lock()
			roster := s.dshbotRoster
			s.mu.Unlock()
			if roster == nil {
				return
			}
			items := make([]pluginrpc.DshbotCatalogItem, len(catalog.Items))
			for i, v := range catalog.Items {
				items[i] = v.DshbotCatalogItem
			}
			roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: pluginrpc.DshbotSettingsView{NS: "dsh-bot", Revision: catalog.Revision, Value: pluginrpc.DshbotCatalogValue{Items: items}}}, false, ep.Origin)
		})
	}
	return assistantui.BotVisibilityModel{State: snapshot, HostLabel: ep.Origin,
		OnRefresh: func() {
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
					if err == nil {
						updateRoster(catalog)
					}
					s.updateAssistantTasks()
				}
			}()
		},
		OnShowHidden: func(on bool) {
			if current() {
				controls.Show(scope, on)
				s.updateAssistantTasks()
			}
		},
		OnReview: func(id, session string) {
			if current() {
				controls.Prepare(scope, id, session, time.Now().UnixMilli())
				s.updateAssistantTasks()
			}
		},
		OnCancel: func() {
			if current() {
				controls.Cancel(scope)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledge: func(on bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, on)
				s.updateAssistantTasks()
			}
		},
		OnAdopt: func(id, session string) {
			if current() && controls.Adopt(scope, id, session) {
				updateRoster(controls.Snapshot(scope).Catalog)
				s.updateAssistantTasks()
			}
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
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotVisibilityReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				result, saveErr := rpc.DshbotSetVisibility(ctx, cmd)
				controls.Finish(ticket, result, saveErr)
				if saveErr == nil {
					updateRoster(result.Catalog)
				}
				s.updateAssistantTasks()
			}()
		},
	}
}
func (s *mainState) observeAssistantBotVisibility(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantBotVisibility.Suspend()
	}
}

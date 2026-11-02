package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) assistantBotDeleteModel(ep harness.Endpoint, sourceID string) assistantui.BotDeleteModel {
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
	controls := &s.assistantBotDeleter
	controls.Bind(scope, binding)
	snapshot := controls.Snapshot(scope)
	current := func() bool {
		s.mu.Lock()
		same := originalAssistant != nil && s.assistant == originalAssistant
		s.mu.Unlock()
		if !same || client == nil || s.assistantRoomClient(ep) != client || !controls.ViewCurrent(scope, snapshot.Generation) {
			return false
		}
		selected := originalAssistant.Snapshot()
		return selected.DshbotID == sourceID && selected.DshbotOrigin == ep.Origin
	}
	authority := func(ctx context.Context) (bool, error) {
		if !current() {
			return false, fmt.Errorf("original Host/account/view expired")
		}
		var out struct {
			Writable bool `json:"writable"`
		}
		err := client.Call(ctx, "settings/describe", nil, &out)
		if !current() {
			return false, fmt.Errorf("original Host/account/view expired")
		}
		return err == nil && out.Writable, err
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
		s.updateAssistantTasks()
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
				s.updateAssistantTasks()
			}
		}()
	}
	return assistantui.BotDeleteModel{State: snapshot, HostLabel: ep.Origin, SourceID: sourceID, OnRefresh: refresh,
		OnReview: func() {
			if current() {
				controls.Prepare(scope)
				s.updateAssistantTasks()
			}
		},
		OnCancel: func() {
			if current() {
				controls.Cancel(scope)
				s.updateAssistantTasks()
			}
		},
		OnIdentity: func(id, name string) {
			if current() {
				controls.ConfirmIdentity(scope, snapshot.Review, id, name)
				s.updateAssistantTasks()
			}
		},
		OnAcknowledge: func(ack bool) {
			if current() {
				controls.Acknowledge(scope, snapshot.Review, ack)
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
				fresh, writable, err := readPlan(ctx)
				if err == nil && (!writable || !pluginrpc.SameDshbotBotDeletePlan(cmd.Plan, fresh)) {
					err = fmt.Errorf("%s", i18n.T(i18n.KeyCmdBotDeleteReviewChanged))
				}
				if err != nil || !current() || !controls.Current(ticket) || !controls.MarkSent(ticket) {
					controls.AbortUnsent(ticket, err)
					s.updateAssistantTasks()
					return
				}
				// Original ID/revision only, one mutation. No separate stop, file removal,
				// retry, session opening or automatic recovery. The Host owns its cascade.
				result, deleteErr := rpc.DshbotDeleteBot(ctx, cmd)
				controls.Finish(ticket, result, deleteErr)
				if deleteErr == nil {
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

// Window hooks cover hidden/non-rendered windows. The frame-end fence catches
// same-frame navigation. Already-dispatched attempts settle only their origin.
func (s *mainState) observeAssistantBotDeleter(rendered bool) {
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	s.mu.Unlock()
	if !rendered || s.assistantRoomClient(ep) == nil {
		s.assistantBotDeleter.Suspend()
	}
}

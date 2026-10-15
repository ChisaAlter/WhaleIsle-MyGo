package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *mainState) assistantTaskClient(ep harness.Endpoint, botID string) *dsh.Client {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if !s.assistantDshbotCurrent(ep, client) {
		return nil
	}
	s.mu.Lock()
	asst := s.assistant
	s.mu.Unlock()
	if asst == nil {
		return nil
	}
	current := asst.Snapshot()
	if current.Mode != assiststore.ModeDshbot || current.DshbotOrigin != ep.Origin || current.DshbotID != botID || botID == "" {
		return nil
	}
	return client
}
func assistantTaskScope(ep harness.Endpoint, client *dsh.Client, botID string) dshbotstore.TaskScope {
	if client == nil {
		return dshbotstore.TaskScope{}
	}
	// An account cookie or authenticated client replacement invalidates the view
	// even when the Host URL and Bot ID happen to be unchanged. Never log this key.
	digest := sha256.Sum256([]byte(fmt.Sprintf("%p\x00%s\x00%s", client, ep.Origin, ep.Cookie)))
	return dshbotstore.TaskScope{HostKey: fmt.Sprintf("%x", digest), BotID: botID}
}
func (s *mainState) updateAssistantTasks() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
func (s *mainState) assistantTasksModel(ep harness.Endpoint, botID string) assistantui.TaskModel {
	client := s.assistantTaskClient(ep, botID)
	scope := assistantTaskScope(ep, client, botID)
	s.assistantTasks.Bind(scope)
	current := func() bool { return client != nil && s.assistantTaskClient(ep, botID) == client }
	refresh := func() {
		if !current() {
			s.assistantTasks.Bind(dshbotstore.TaskScope{})
			return
		}
		ticket, ok := s.assistantTasks.BeginLoad(scope)
		if !ok {
			return
		}
		s.updateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var view *pluginrpc.DshbotTaskView
			var err error
			if current() {
				view, err = s.pluginRPCClient(ep).DshbotTasks(ctx, botID)
			} else {
				err = fmt.Errorf("Host or Bot selection changed; refresh again")
			}
			if !current() {
				return
			}
			if s.assistantTasks.FinishLoad(ticket, view, err) {
				s.updateAssistantTasks()
			}
		}()
	}
	return assistantui.TaskModel{
		State:     s.assistantTasks.Snapshot(scope),
		OnRefresh: refresh,
		OnSelect: func(id string) {
			if current() {
				s.assistantTasks.Select(scope, id)
				s.updateAssistantTasks()
			}
		},
		OnReview: func(action, id string) {
			if current() {
				s.assistantTasks.Prepare(scope, action, id)
				s.updateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantTasks.Dismiss(scope)
				s.updateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantTasks.BeginAction(scope)
			s.updateAssistantTasks()
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				var result pluginrpc.DshbotTaskResult
				var err error
				// No bot/open, prompt, automatic retry, or optimistic task state mutation.
				if current() {
					result, err = s.pluginRPCClient(ep).DshbotControlTask(ctx, command)
				} else {
					err = fmt.Errorf("Host or Bot selection changed before the action")
				}
				if !current() {
					return
				}
				if s.assistantTasks.FinishAction(ticket, result, err) {
					s.updateAssistantTasks()
				}
			}()
		},
	}
}

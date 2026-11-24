package assistantctl

import (
	"context"
	"fmt"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
)

func (s *Service) taskClient(ep harness.Endpoint, botID string) *dsh.Client {
	client := s.safeHost().CurrentClient()
	if !s.DshbotCurrent(ep, client) {
		return nil
	}
	asst := s.AssistantStore()
	if asst == nil {
		return nil
	}
	current := asst.Snapshot().Value
	if current.Mode != assiststore.ModeDshbot || current.DshbotOrigin != ep.Origin || current.DshbotID != botID || botID == "" {
		return nil
	}
	return client
}

func taskScope(ep harness.Endpoint, client *dsh.Client, botID string) dshbotstore.TaskScope {
	if client == nil {
		return dshbotstore.TaskScope{}
	}
	// An account cookie or authenticated client replacement invalidates the view
	// even when the Host URL and Bot ID happen to be unchanged. Never log this key.
	return dshbotstore.TaskScope{
		HostKey: endpointScope(clientBinding(client), ep.Origin, ep.Cookie),
		BotID:   botID,
	}
}

// TasksModel builds the Bot task control UI model bound to the current endpoint and Bot.
func (s *Service) TasksModel(ep harness.Endpoint, botID string) assistantui.TaskModel {
	if s == nil {
		return assistantui.TaskModel{}
	}
	client := s.taskClient(ep, botID)
	scope := taskScope(ep, client, botID)
	s.assistantTasks.Bind(scope)
	current := func() bool { return client != nil && s.taskClient(ep, botID) == client }
	rpc := s.safeHost().PluginRPCClient(ep)
	refresh := func() {
		if !current() {
			s.assistantTasks.Bind(dshbotstore.TaskScope{})
			return
		}
		ticket, ok := s.assistantTasks.BeginLoad(scope)
		if !ok {
			return
		}
		s.UpdateAssistantTasks()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var view *pluginrpc.DshbotTaskView
			var err error
			if current() {
				view, err = rpc.DshbotTasks(ctx, botID)
			} else {
				err = fmt.Errorf("Host or Bot selection changed; refresh again")
			}
			if !current() {
				return
			}
			if s.assistantTasks.FinishLoad(ticket, view, err) {
				s.UpdateAssistantTasks()
			}
		}()
	}
	return assistantui.TaskModel{
		State:     s.assistantTasks.Snapshot(scope),
		OnRefresh: refresh,
		OnSelect: func(id string) {
			if current() {
				s.assistantTasks.Select(scope, id)
				s.UpdateAssistantTasks()
			}
		},
		OnReview: func(action, id string) {
			if current() {
				s.assistantTasks.Prepare(scope, action, id)
				s.UpdateAssistantTasks()
			}
		},
		OnDismiss: func() {
			if current() {
				s.assistantTasks.Dismiss(scope)
				s.UpdateAssistantTasks()
			}
		},
		OnConfirm: func() {
			if !current() {
				return
			}
			ticket, command, ok := s.assistantTasks.BeginAction(scope)
			s.UpdateAssistantTasks()
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
					result, err = rpc.DshbotControlTask(ctx, command)
				} else {
					err = fmt.Errorf("Host or Bot selection changed before the action")
				}
				if !current() {
					return
				}
				if s.assistantTasks.FinishAction(ticket, result, err) {
					s.UpdateAssistantTasks()
				}
			}()
		},
	}
}

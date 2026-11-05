package session

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// GoalDockAction captures one visible goal/CAS reference. It returns the actual
// result to the local editor; legacy fire-and-forget goal APIs remain usable.
func (s *Store) GoalDockAction(goalID string, revision float64) func(context.Context, string, string) error {
	s.mu.Lock()
	client, sessionID, generation := s.client, s.selectedID, s.followGeneration
	var endpoint dsh.Endpoint
	if client != nil {
		endpoint = client.Endpoint
	}
	s.mu.Unlock()
	return func(ctx context.Context, action, objective string) error {
		s.mu.Lock()
		current := client != nil && s.sessionResultCurrentLocked(ctx, client, sessionID, generation) && client.Endpoint == endpoint && s.planBoard.Goal != nil && s.planBoard.Goal.ID == goalID && s.planBoard.Goal.Revision == revision
		s.mu.Unlock()
		if !current {
			return fmt.Errorf("goal owner changed")
		}
		if action != "edit" && action != "pause" && action != "resume" && action != "clear" {
			return fmt.Errorf("unsupported goal action")
		}
		args := map[string]any{"agentId": sessionID, "ref": api.GoalRef{Id: goalID, Revision: revision}}
		if action == "edit" {
			objective = strings.TrimSpace(objective)
			if objective == "" {
				return fmt.Errorf("empty objective")
			}
			args["request"] = api.EditGoalRequest{Objective: &objective}
		}
		var out api.GoalView
		err := client.Call(ctx, "goals/"+action, args, &out)
		s.mu.Lock()
		if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) || client.Endpoint != endpoint || s.planBoard.Goal == nil || s.planBoard.Goal.ID != goalID {
			s.mu.Unlock()
			return context.Canceled
		}
		if err == nil {
			if action == "clear" {
				s.planBoard.Goal = nil
			} else if out.Id != "" {
				s.applyGoalView(&out)
			} else {
				err = fmt.Errorf("goal action returned no goal state")
			}
		}
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "goal " + action
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return err
	}
}

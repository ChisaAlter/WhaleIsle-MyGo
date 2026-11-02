package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// CreateGoal calls goals/create for the selected session.
func (s *Store) CreateGoal(ctx context.Context, objective string) {
	objective = strings.TrimSpace(objective)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" || objective == "" {
		return
	}
	go s.runCreateGoal(ctx, client, sessionID, generation, objective)
}

func (s *Store) runCreateGoal(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, objective string) {
	var out api.CreateGoalResult
	err := client.Call(ctx, "goals/create", map[string]any{
		"agentId": sessionID,
		"request": api.CreateGoalRequest{Objective: objective},
	}, &out)
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.lastAction = err.Error()
	} else {
		s.planBoard.goalReadToken = nil
		s.lastAction = "goal created"
		s.planBoard.Goal = &PlanGoal{
			ID:        out.Ref.Id,
			Revision:  out.Ref.Revision,
			Objective: objective,
			Phase:     "active",
		}
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

// EditGoal calls goals/edit for the current board goal.
func (s *Store) EditGoal(ctx context.Context, objective string) {
	objective = strings.TrimSpace(objective)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	var ref *api.GoalRef
	if s.planBoard.Goal != nil && s.planBoard.Goal.ID != "" {
		ref = &api.GoalRef{Id: s.planBoard.Goal.ID, Revision: s.planBoard.Goal.Revision}
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" || ref == nil || objective == "" {
		return
	}
	go s.runEditGoal(ctx, client, sessionID, generation, objective, ref)
}

func (s *Store) runEditGoal(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, objective string, ref *api.GoalRef) {
	var out api.GoalView
	err := client.Call(ctx, "goals/edit", map[string]any{
		"agentId": sessionID,
		"ref":     *ref,
		"request": api.EditGoalRequest{Objective: &objective},
	}, &out)
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.lastAction = err.Error()
	} else {
		s.lastAction = "goal edited"
		s.applyGoalView(&out)
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

// CompleteGoal calls goals/complete for the current board goal.
func (s *Store) CompleteGoal(ctx context.Context) {
	s.goalAction(ctx, "goals/complete", "goal completed", true)
}

// ClearGoal calls goals/clear for the current board goal.
func (s *Store) ClearGoal(ctx context.Context) {
	s.goalAction(ctx, "goals/clear", "goal cleared", true)
}

// PauseGoal calls goals/pause for the current board goal.
func (s *Store) PauseGoal(ctx context.Context) {
	s.goalAction(ctx, "goals/pause", "goal paused", false)
}

// ResumeGoal calls goals/resume for the current board goal.
func (s *Store) ResumeGoal(ctx context.Context) {
	s.goalAction(ctx, "goals/resume", "goal resumed", false)
}

// RefreshGoal calls goals/get and folds the view into the plan board.
func (s *Store) RefreshGoal(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return
	}
	go s.runRefreshGoal(ctx, client, sessionID, generation)
}

func (s *Store) runRefreshGoal(ctx context.Context, client *dsh.Client, sessionID string, generation uint64) {
	s.runRefreshGoalForOwner(ctx, client, sessionID, generation, nil)
}

// A newly selected workspace session owns its initial goal read by Host and
// selection generation as well as by client pointer. A reconnect can reuse the
// same client; neither the old read nor its result belongs to the new Host.
func (s *Store) runRefreshGoalForOwner(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, owner *workspaceSelectedOwner) {
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) || (owner != nil && !owner.currentLocked(s)) {
		s.mu.Unlock()
		return
	}
	token := new(int)
	s.planBoard.goalReadToken = token
	s.mu.Unlock()
	var out api.GoalView
	err := client.Call(ctx, "goals/get", map[string]any{"agentId": sessionID}, &out)
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) || (owner != nil && !owner.currentLocked(s)) || s.planBoard.goalReadToken != token {
		s.mu.Unlock()
		return
	}
	s.planBoard.goalReadToken = nil
	if err != nil {
		s.goalReadError = goalReadErrorState{client: client, sessionID: sessionID, generation: generation, message: "goals/get: " + err.Error()}
	} else {
		s.goalReadError = goalReadErrorState{}
		// goals/get returns undefined when this live agent has no current goal.
		// Empty success is ordinary optional state, never a global action/error.
		if out.Id != "" {
			s.applyGoalView(&out)
		} else {
			s.planBoard.Goal = nil
		}
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) goalAction(ctx context.Context, method, okAction string, clear bool) {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	generation := s.followGeneration
	var ref *api.GoalRef
	if s.planBoard.Goal != nil && s.planBoard.Goal.ID != "" {
		ref = &api.GoalRef{Id: s.planBoard.Goal.ID, Revision: s.planBoard.Goal.Revision}
	}
	s.mu.Unlock()
	if client == nil || sessionID == "" || ref == nil {
		return
	}
	go s.runGoalAction(ctx, client, sessionID, generation, method, okAction, clear, ref)
}

func (s *Store) runGoalAction(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, method, okAction string, clear bool, ref *api.GoalRef) {
	var out api.GoalView
	err := client.Call(ctx, method, map[string]any{
		"agentId": sessionID,
		"ref":     *ref,
	}, &out)
	s.mu.Lock()
	if !s.sessionResultCurrentLocked(ctx, client, sessionID, generation) {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.lastAction = err.Error()
	} else {
		s.planBoard.goalReadToken = nil
		s.lastAction = okAction
		if clear {
			s.planBoard.Goal = nil
		} else if out.Id != "" {
			s.applyGoalView(&out)
		} else {
			s.lastAction = "goal action returned no goal state"
		}
	}
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applyGoalView(v *api.GoalView) {
	if v == nil || v.Id == "" {
		return
	}
	s.planBoard.goalReadToken = nil
	s.planBoard.Goal = &PlanGoal{
		ID:         v.Id,
		Revision:   v.Revision,
		Objective:  v.Objective,
		Phase:      v.Phase,
		Activation: v.Activation,
	}
	if v.BlockedReason != nil {
		s.planBoard.Goal.BlockedReason = PlanGoalBlockedReason{Code: v.BlockedReason.Code, Message: v.BlockedReason.Message}
	}
}

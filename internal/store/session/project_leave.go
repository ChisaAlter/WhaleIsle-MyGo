package session

import (
	"context"
	"errors"
)

// LeaveProjectLead navigates the native main view to an ordinary blank session
// in the same registered workspace (or scratch). It neither deletes Project data
// nor stops the Lead. The switch may proceed only after the new follow is ready.
func (s *Store) LeaveProjectLead(ctx context.Context, excludeIDs ...string) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	var directory string
	lead := false
	for _, item := range s.items {
		if item.SessionID == s.selectedID && item.Presentation != nil && item.Presentation.Owner == "project" {
			lead = true
			directory = item.Presentation.WorkingDirectory
			if directory == "" {
				directory = item.Cwd
			}
			break
		}
	}
	if !lead {
		s.mu.Unlock()
		return nil
	}
	if s.client == nil || s.mux == nil {
		s.mu.Unlock()
		return ErrNoClient
	}
	host, source, follow := s.eventsGeneration, s.selectedID, s.followGeneration
	// Scratch reuse is shared with ordinary creation. Never propose a Project
	// Lead there, even when its Host summary still calls it blank.
	for _, item := range s.items {
		if item.Presentation != nil && item.Presentation.Owner == "project" {
			excludeIDs = append(excludeIDs, item.SessionID)
		}
	}
	workspaceID := "::scratch"
	for id, workspace := range s.workspaceMemberships {
		if hostPathKey(workspace.Path) == hostPathKey(directory) {
			workspaceID = id
			break
		}
	}
	s.mu.Unlock()
	target, err := s.CaptureWorkspaceSessionTarget(host, workspaceID, excludeIDs...)
	if err != nil {
		return err
	}
	if target.SourceSessionID() != source || target.SourceFollowGeneration() != follow {
		return errors.New("Project Lead navigation changed before leaving")
	}
	id, err := target.Create(ctx)
	if err != nil {
		return err
	}
	if !target.Select(id) {
		return errors.New("Project Lead navigation changed before leaving")
	}
	return s.waitProjectLeaveReady(ctx, host, id, follow+1)
}

func (s *Store) waitProjectLeaveReady(ctx context.Context, host uint64, id string, expectedFollow ...uint64) error {
	for {
		s.mu.Lock()
		if s.eventsGeneration != host || s.selectedID != id || len(expectedFollow) > 0 && s.followGeneration != expectedFollow[0] {
			s.mu.Unlock()
			return errors.New("Project Lead navigation changed while awaiting the new session")
		}
		for _, item := range s.items {
			if item.SessionID == id && item.Presentation != nil && item.Presentation.Owner == "project" {
				s.mu.Unlock()
				return errors.New("Project Lead exit selected another Project")
			}
		}
		if s.historyError != "" {
			message := s.historyError
			s.mu.Unlock()
			return errors.New(message)
		}
		if s.historyReady && !s.historyLoading {
			s.mu.Unlock()
			return nil
		}
		if s.viewChanged == nil {
			s.viewChanged = make(chan struct{})
		}
		changed := s.viewChanged
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (s *Store) ProjectLeadSelected() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		if item.SessionID == s.selectedID {
			return item.Presentation != nil && item.Presentation.Owner == "project"
		}
	}
	return false
}

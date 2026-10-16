package main

import (
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/frame"
)

func (s *mainState) openScheduleTask(row schedstore.Entry) {
	s.mu.Lock()
	store := s.schedule
	s.mu.Unlock()
	if store == nil {
		return
	}
	snap := store.Snapshot()
	if !snap.ClientReady || row.Origin != snap.Origin {
		return
	}
	for _, current := range snap.Rows {
		if current.ID == row.ID && current.SessionID == row.SessionID {
			s.mu.Lock()
			if s.layout.Workspace == nil {
				s.layout.Workspace = &frame.WorkspacePanes{}
			}
			viewer := s.sessions.Snapshot().SelectedID
			s.layout.Workspace.OpenScheduleTask(viewer, current)
			s.filesPaneOpen, s.rightPaneTab = true, "schedule-task"
			s.mu.Unlock()
			s.selectMainPanel("")
			return
		}
	}
}

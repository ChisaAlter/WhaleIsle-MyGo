package main

import (
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func (s *mainState) openScheduleTask(row schedstore.Entry) {
	s.mu.Lock()
	store := s.schedule
	sessions := s.sessions
	s.mu.Unlock()
	if store == nil || sessions == nil {
		return
	}
	snap := store.Snapshot()
	if !snap.ClientReady || row.Origin != snap.Origin {
		return
	}
	viewer := sessions.Snapshot().SelectedID
	for _, current := range snap.Rows {
		if current.ID == row.ID && current.SessionID == row.SessionID {
			s.showScheduleTask(current, store, snap.ConnectionEpoch, viewer, false)
			return
		}
	}
	registered, settled := store.ReceiptCatalogState(row)
	if !registered || settled {
		return
	}
	owner := sessions.Snapshot()
	if !owner.ClientReady || owner.SelectedID != row.SessionID {
		return
	}
	// The fallback is rebuilt from the owning Session's completed root receipt,
	// never from a caller-supplied title, rule, or fabricated receipt payload.
	for _, receipts := range session.ScheduleCards(owner.Messages, owner.SelectedID) {
		for _, receipt := range receipts {
			if receipt.ID == row.ID && receipt.SessionID == row.SessionID {
				receipt.Origin = snap.Origin
				s.showScheduleTask(receipt, store, snap.ConnectionEpoch, viewer, true)
				return
			}
		}
	}
}

func (s *mainState) showScheduleTask(row schedstore.Entry, store *schedstore.Store, epoch uint64, viewer string, receipt bool) {
	s.mu.Lock()
	if s.schedule != store || s.sessions == nil || s.sessions.Snapshot().SelectedID != viewer {
		s.mu.Unlock()
		return
	}
	current := store.Snapshot()
	if !current.ClientReady || current.ConnectionEpoch != epoch || current.Origin != row.Origin {
		s.mu.Unlock()
		return
	}
	if receipt {
		_, settled := store.ReceiptCatalogState(row)
		if settled {
			found := false
			for _, latest := range current.Rows {
				if latest.ID == row.ID && latest.SessionID == row.SessionID {
					row, found = latest, true
					break
				}
			}
			if !found {
				s.mu.Unlock()
				return
			}
		}
	}
	if s.layout.Workspace == nil {
		s.layout.Workspace = &frame.WorkspacePanes{}
	}
	s.layout.Workspace.OpenScheduleTask(viewer, row)
	s.filesPaneOpen, s.rightPaneTab = true, "schedule-task"
	s.mu.Unlock()
	s.selectMainPanel("")
}

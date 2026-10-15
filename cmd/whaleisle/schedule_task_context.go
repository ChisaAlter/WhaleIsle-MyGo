package main

import schedstore "whaleisle/internal/store/schedule"

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
			s.schedulePanel.OpenTask(current, snap.ConnectionEpoch)
			s.selectMainPanel("schedule")
			return
		}
	}
}

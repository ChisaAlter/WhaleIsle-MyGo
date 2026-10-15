package schedule

import "context"

// EnsureContext starts the shared catalog once and refreshes after newly seen
// creation receipts. It issues no mutations and shares schedule/changed updates.
func (s *Store) EnsureContext(ctx context.Context, sessionID string, receiptIDs []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.client == nil {
		s.mu.Unlock()
		return
	}
	refresh := !s.requested
	if s.receiptKeys == nil {
		s.receiptKeys = map[string]bool{}
	}
	for _, id := range receiptIDs {
		key := s.client.Endpoint.Origin + "\x00" + sessionID + "\x00" + id
		if !s.receiptKeys[key] {
			s.receiptKeys[key] = true
			refresh = true
		}
	}
	s.mu.Unlock()
	if refresh {
		s.Refresh(ctx, sessionID)
	}
}

func ActiveForSession(snap ViewState, id string) []Entry {
	if !snap.ClientReady || snap.Err != "" {
		return nil
	}
	var rows []Entry
	for _, row := range snap.Rows {
		if row.SessionID == id && row.Status == "active" {
			rows = append(rows, row)
		}
	}
	return FilterEntries(rows, "", "active")
}

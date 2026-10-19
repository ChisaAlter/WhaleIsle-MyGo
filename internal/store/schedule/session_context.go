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
	if s.receiptReads == nil {
		s.receiptReads = map[string]uint64{}
	}
	for _, id := range receiptIDs {
		key := s.client.Endpoint.Origin + "\x00" + sessionID + "\x00" + id
		if !s.receiptKeys[key] {
			s.receiptKeys[key] = true
			s.receiptReads[key] = s.catalogRequest
			refresh = true
		}
	}
	s.mu.Unlock()
	if refresh {
		s.Refresh(ctx, sessionID)
	}
}

// ReceiptCatalogState reports whether this receipt was registered and whether
// an all-session read issued after this
// receipt first appeared has succeeded. An earlier in-flight read cannot prove
// that a newly created task is absent; a failed read keeps the creation receipt.
func (s *Store) ReceiptCatalogState(row Entry) (registered, settled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || row.Origin != s.client.Endpoint.Origin {
		return false, false
	}
	key := row.Origin + "\x00" + row.SessionID + "\x00" + row.ID
	required, registered := s.receiptReads[key]
	return registered, registered && s.catalogReady && s.catalogSettled > required
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

package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"whaleisle/internal/dsh/api"
)

// HistoryPage follows the Host's opaque delivery cursor without reconstructing missing receipts.
func (s *Store) HistoryPage(ctx context.Context, row Entry, before string) (string, string, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || (row.Origin != "" && row.Origin != client.Endpoint.Origin) {
		return "", "", fmt.Errorf("not connected to the task's Host")
	}
	if row.SessionID == "" || row.ID == "" {
		return "", "", fmt.Errorf("missing owning Session or task")
	}
	req := api.ScheduleDeliveryHistoryRequest{Id: row.ID, SessionId: row.SessionID, Limit: 20}
	if before != "" {
		req.Before = &before
	}
	var out api.ScheduleDeliveryHistoryResult
	if err := client.Call(ctx, "schedule/history", map[string]any{"request": req}, &out); err != nil {
		return "", "", err
	}
	if err := resultError(out); err != nil {
		return "", "", err
	}
	var result struct {
		NextBefore string `json:"nextBefore"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", "", err
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	return string(pretty), result.NextBefore, nil
}

package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"whaleisle/internal/dsh/api"
)

// DeliveryRecord is an acknowledged inbox delivery, not an execution result.
// A missing prompt is valid for legacy receipts and must not use today's prompt.
type DeliveryRecord struct {
	MessageID   string  `json:"messageId"`
	ScheduledAt string  `json:"scheduledAt"`
	DeliveredAt string  `json:"deliveredAt"`
	Prompt      *string `json:"prompt,omitempty"`
}
type DeliveryRetention struct {
	Days    int `json:"days"`
	Records int `json:"records"`
}
type DeliveryPage struct {
	ID                        string             `json:"id"`
	Records                   []DeliveryRecord   `json:"records"`
	NextBefore                string             `json:"nextBefore,omitempty"`
	EarlierRecordsUnavailable bool               `json:"earlierRecordsUnavailable"`
	EarlierRecordsPruned      bool               `json:"earlierRecordsPruned"`
	Retention                 *DeliveryRetention `json:"retention,omitempty"`
}

// MergeDeliveryPages retains newest-first Host order and immutable receipt
// identity. Cursor/retention metadata belongs to the newly fetched older page.
func MergeDeliveryPages(previous, next DeliveryPage) DeliveryPage {
	merged := next
	merged.Records = make([]DeliveryRecord, 0, len(previous.Records)+len(next.Records))
	seen := map[string]bool{}
	for _, records := range [][]DeliveryRecord{previous.Records, next.Records} {
		for _, record := range records {
			if !seen[record.MessageID] {
				seen[record.MessageID] = true
				merged.Records = append(merged.Records, record)
			}
		}
	}
	return merged
}

func DecodeDeliveryPage(raw json.RawMessage, id string) (DeliveryPage, error) {
	var out DeliveryPage
	if err := resultError(raw); err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if out.ID != id || out.Records == nil {
		return DeliveryPage{}, fmt.Errorf("invalid delivery history for task %s", id)
	}
	for _, r := range out.Records {
		if r.MessageID == "" || r.ScheduledAt == "" || r.DeliveredAt == "" {
			return DeliveryPage{}, fmt.Errorf("incomplete saved delivery")
		}
	}
	return MergeDeliveryPages(DeliveryPage{}, out), nil
}

// DeliveryHistory follows the Host's opaque cursor without reconstructing receipts.
func (s *Store) DeliveryHistory(ctx context.Context, row Entry, before string) (DeliveryPage, error) {
	s.mu.Lock()
	client := s.client
	epoch := s.connectionEpoch
	s.mu.Unlock()
	if client == nil || (row.Origin != "" && row.Origin != client.Endpoint.Origin) {
		return DeliveryPage{}, fmt.Errorf("not connected to the task's Host")
	}
	if row.SessionID == "" || row.ID == "" {
		return DeliveryPage{}, fmt.Errorf("missing owning Session or task")
	}
	req := api.ScheduleDeliveryHistoryRequest{Id: row.ID, SessionId: row.SessionID, Limit: 20}
	if before != "" {
		req.Before = &before
	}
	var raw api.ScheduleDeliveryHistoryResult
	if err := client.Call(ctx, "schedule/history", map[string]any{"request": req}, &raw); err != nil {
		return DeliveryPage{}, err
	}
	s.mu.Lock()
	current := s.client == client && s.connectionEpoch == epoch
	s.mu.Unlock()
	if !current {
		return DeliveryPage{}, fmt.Errorf("Host connection changed; reload task history")
	}
	return DecodeDeliveryPage(raw, row.ID)
}

// HistoryPage keeps the copy/export API while native rendering consumes typed records.
func (s *Store) HistoryPage(ctx context.Context, row Entry, before string) (string, string, error) {
	page, err := s.DeliveryHistory(ctx, row, before)
	if err != nil {
		return "", "", err
	}
	raw, _ := json.MarshalIndent(page, "", "  ")
	return string(raw), page.NextBefore, nil
}

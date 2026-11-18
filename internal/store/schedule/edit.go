package schedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"whaleisle/internal/dsh/api"
)

// RuleSnapshot excludes catalog/delivery metadata from the exact CAS record.
func RuleSnapshot(raw json.RawMessage) (json.RawMessage, error) {
	var record map[string]json.RawMessage
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	kind := ""
	_ = json.Unmarshal(record["kind"], &kind)
	keys := []string{"id", "kind", "title", "prompt", "scheduledAt"}
	switch kind {
	case "at":
	case "after":
		keys = append(keys, "afterSeconds")
	case "every":
		keys = append(keys, "everySeconds")
	case "daily":
		keys = append(keys, "time", "timeZone")
	case "weekly":
		keys = append(keys, "time", "timeZone", "weekdays")
	case "cron":
		keys = append(keys, "expression", "timeZone")
	default:
		return nil, fmt.Errorf("unsupported schedule kind %q", kind)
	}
	out := map[string]json.RawMessage{}
	for _, key := range keys {
		if len(record[key]) == 0 {
			return nil, fmt.Errorf("missing schedule field %s", key)
		}
		out[key] = record[key]
	}
	return json.Marshal(out)
}

// scheduleEditRequest is the pinned runtime's edit envelope: one action against
// one task, carrying the updatedAt value the editor read as its concurrency
// baseline. It replaced the rc.4 schedule/update + schedule/delete pair, whose
// request carried a full rule snapshot instead.
func scheduleEditRequest(row Entry, action string, fields map[string]any) map[string]any {
	request := map[string]any{"id": row.ID, "action": action, "expectedUpdatedAt": expectedUpdatedAt(row.Raw)}
	for key, value := range fields {
		request[key] = value
	}
	return request
}

// expectedUpdatedAt is the list record's own updatedAt, or null when the record
// carries none, which the Host accepts as "no baseline".
func expectedUpdatedAt(raw json.RawMessage) json.RawMessage {
	var record map[string]json.RawMessage
	if json.Unmarshal(raw, &record) == nil {
		if value := record["updatedAt"]; len(value) > 0 {
			return value
		}
	}
	return json.RawMessage("null")
}

// ResultError retains the Host recovery code without parsing display text.
type ResultError struct {
	Code    string
	Message string
}

func (e *ResultError) Error() string { return e.Code + ": " + e.Message }

func resultError(raw json.RawMessage) error {
	var out struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("invalid schedule result: %w", err)
	}
	if out.Code != "" {
		return &ResultError{Code: out.Code, Message: out.Message}
	}
	return nil
}

// Update commits against the record captured when editing began, never a newer list.
func (s *Store) Update(ctx context.Context, row Entry, title, prompt *string, change json.RawMessage) error {
	expected, err := RuleSnapshot(row.Raw)
	if err != nil {
		return err
	}
	if strings.TrimSpace(row.SessionID) == "" || row.ID == "" {
		return fmt.Errorf("missing owning Session or task")
	}
	s.mu.Lock()
	client := s.client
	epoch := s.connectionEpoch
	if s.busy || client == nil || (row.Origin != "" && row.Origin != client.Endpoint.Origin) {
		s.mu.Unlock()
		return fmt.Errorf("not connected or another schedule write is pending")
	}
	s.busy = true
	s.lastAction = ""
	s.mu.Unlock()
	s.fire()
	patch := map[string]any{}
	if title != nil {
		patch["title"] = *title
	}
	if prompt != nil {
		patch["prompt"] = *prompt
	}
	if trimmed := bytes.TrimSpace(change); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		patch["timing"] = change
	}
	var out api.ScheduleEditResult
	err = client.Call(ctx, "schedule/edit", map[string]any{"request": scheduleEditRequest(row, "update", map[string]any{"patch": patch})}, &out)
	if err == nil {
		err = resultError(out)
	}
	var committed json.RawMessage
	if err == nil {
		var result struct {
			Record json.RawMessage `json:"record"`
		}
		_ = json.Unmarshal(out, &result)
		committed = result.Record
		if len(result.Record) == 0 {
			err = fmt.Errorf("schedule update returned no committed record")
		}
	}
	s.mu.Lock()
	if s.client != client || s.connectionEpoch != epoch {
		s.mu.Unlock()
		return fmt.Errorf("Host connection changed; refresh to verify the result")
	}
	s.busy = false
	if err == nil {
		s.lastAction = "Schedule saved"
		for i := range s.rows {
			if s.rows[i].ID == row.ID && s.rows[i].SessionID == row.SessionID {
				currentRule, snapshotErr := RuleSnapshot(s.rows[i].Raw)
				if snapshotErr != nil || !bytes.Equal(currentRule, expected) {
					// A catalog already observed a later rule while this save was
					// in flight. Its newer CAS baseline must remain authoritative.
					continue
				}
				// The committed rule owns its new instant/zone as well as text.
				// Preserve catalog-only ownership and lifecycle metadata.
				var record, metadata map[string]json.RawMessage
				_ = json.Unmarshal(committed, &record)
				_ = json.Unmarshal(s.rows[i].Raw, &metadata)
				for _, key := range []string{"sessionId", "status", "lastDelivery"} {
					if len(record[key]) == 0 && len(metadata[key]) != 0 {
						record[key] = metadata[key]
					}
				}
				merged, _ := json.Marshal(record)
				fresh := parseScheduleList([]json.RawMessage{merged})[0]
				fresh.Origin, fresh.SessionID, fresh.Status = s.rows[i].Origin, s.rows[i].SessionID, s.rows[i].Status
				s.rows[i] = fresh
			}
		}
	}
	current := s.client == client
	sid := s.sessionID
	s.mu.Unlock()
	s.fire()
	var failure *ResultError
	refresh := err == nil || (errors.As(err, &failure) &&
		(failure.Code == "schedule_conflict" || failure.Code == "schedule_ended" || failure.Code == "schedule_not_found"))
	if refresh && current {
		s.Refresh(ctx, sid)
	}
	return err
}

func (s *Store) DeleteChecked(ctx context.Context, sessionID, id string) error {
	return s.DeleteEntry(ctx, Entry{SessionID: sessionID, ID: id})
}
func (s *Store) DeleteEntry(ctx context.Context, row Entry) error {
	sessionID, id := row.SessionID, row.ID
	if sessionID == "" || id == "" {
		return fmt.Errorf("missing owning Session or task")
	}
	s.mu.Lock()
	client := s.client
	epoch := s.connectionEpoch
	if s.busy || client == nil || (row.Origin != "" && row.Origin != client.Endpoint.Origin) {
		s.mu.Unlock()
		return fmt.Errorf("not connected or another schedule write is pending")
	}
	s.busy = true
	s.lastAction = ""
	s.mu.Unlock()
	s.fire()
	var out api.ScheduleEditResult
	err := client.Call(ctx, "schedule/edit", map[string]any{"request": scheduleEditRequest(row, "delete", nil)}, &out)
	if err == nil {
		err = resultError(out)
	}
	var failure *ResultError
	alreadyAbsent := false
	if errors.As(err, &failure) && failure.Code == "schedule_not_found" {
		// Already absent is a settled deletion; the catalog still has to confirm it.
		err = nil
		alreadyAbsent = true
		out = json.RawMessage(`{"deleted":true}`)
	}
	if err == nil {
		var result struct {
			Deleted bool `json:"deleted"`
		}
		_ = json.Unmarshal(out, &result)
		if !result.Deleted {
			err = fmt.Errorf("schedule not deleted; refresh its current status")
		}
	}
	s.mu.Lock()
	if s.client != client || s.connectionEpoch != epoch {
		s.mu.Unlock()
		return fmt.Errorf("Host connection changed; refresh to verify the result")
	}
	s.busy = false
	if err == nil {
		s.lastAction = "Schedule deleted"
	}
	outcome := "failed"
	if err == nil {
		outcome = "deleted"
		if alreadyAbsent {
			outcome = "alreadyAbsent"
		}
	}
	s.publishDeletionLocked(outcome)
	current := s.client == client
	sid := s.sessionID
	s.mu.Unlock()
	s.fire()
	if err == nil && current {
		s.Refresh(ctx, sid)
	}
	return err
}

package session

import (
	"encoding/json"
	"strings"
	schedstore "whaleisle/internal/store/schedule"
)

// ScheduleCards derives root schedule_create receipts at their own completed
// turn. It reads persisted events so history replay needs no extra Host calls.
func ScheduleCards(messages []Message, sessionID string) map[float64][]schedstore.Entry {
	type event struct {
		Type string
		Seq  float64
		Data struct {
			Turn         float64
			CallID, Name string
			Message      struct {
				IsError bool
				Source  struct{ CallID string }
				Content []struct{ Type, Text string }
			}
		}
	}
	var events []event
	ends := map[float64]float64{}
	starts := map[float64]bool{}
	calls := map[float64]map[string]bool{}
	for _, m := range messages {
		var e event
		if m.Streaming || json.Unmarshal(m.Raw, &e) != nil || e.Data.Turn == 0 {
			continue
		}
		events = append(events, e)
		if e.Type == "turn/start" {
			starts[e.Data.Turn] = true
		}
		if e.Type == "turn/end" {
			ends[e.Data.Turn] = e.Seq
		}
		if e.Type == "tool/call" && e.Data.Name == "schedule_create" {
			if calls[e.Data.Turn] == nil {
				calls[e.Data.Turn] = map[string]bool{}
			}
			calls[e.Data.Turn][e.Data.CallID] = true
		}
	}
	out := map[float64][]schedstore.Entry{}
	seen := map[float64]map[string]bool{}
	for _, e := range events {
		end, closed := ends[e.Data.Turn]
		msg := e.Data.Message
		if !closed || !starts[e.Data.Turn] || e.Seq > end || e.Type != "tool/result" || msg.IsError || !calls[e.Data.Turn][msg.Source.CallID] || len(msg.Content) != 1 || msg.Content[0].Type != "text" {
			continue
		}
		if seen[e.Data.Turn] == nil {
			seen[e.Data.Turn] = map[string]bool{}
		}
		if seen[e.Data.Turn][msg.Source.CallID] {
			continue
		}
		row, ok := scheduleReceipt(msg.Content[0].Text)
		if !ok {
			continue
		}
		row.SessionID = sessionID
		out[end] = append(out[end], row)
		seen[e.Data.Turn][msg.Source.CallID] = true
	}
	return out
}

func scheduleReceipt(text string) (schedstore.Entry, bool) {
	var v struct {
		ID, Title, Prompt, Kind    string
		ScheduledAt                *string
		AfterSeconds, EverySeconds *float64
		Time, TimeZone, Expression *string
		Weekdays                   *[]float64
	}
	if json.Unmarshal([]byte(text), &v) != nil || v.ID == "" || v.Prompt == "" || v.ScheduledAt == nil {
		return schedstore.Entry{}, false
	}
	valid := false
	switch v.Kind {
	case "at":
		valid = true
	case "after":
		valid = v.AfterSeconds != nil
	case "every":
		valid = v.EverySeconds != nil
	case "daily":
		valid = v.Time != nil && v.TimeZone != nil
	case "weekly":
		valid = v.Time != nil && v.TimeZone != nil && v.Weekdays != nil
	case "cron":
		valid = v.Expression != nil && v.TimeZone != nil
	}
	if !valid {
		return schedstore.Entry{}, false
	}
	title := strings.TrimSpace(v.Title)
	if title == "" {
		title = strings.TrimSpace(strings.SplitN(v.Prompt, "\n", 2)[0])
	}
	if title == "" {
		return schedstore.Entry{}, false
	}
	zone := ""
	if v.TimeZone != nil {
		zone = *v.TimeZone
	}
	return schedstore.Entry{ID: v.ID, Title: title, Prompt: v.Prompt, Status: "active", ScheduledAt: *v.ScheduledAt, TimeZone: zone, Raw: json.RawMessage(text)}, true
}

package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"whaleisle/internal/desktop/botnotify"
)

// DshbotNotificationSnapshot reads exactly the original contacts catalog items,
// including hidden rows/rooms. It derives one explicit-ID batch after describe;
// an empty catalog never falls through to bot/activity's implicit all-items mode.
func (c *Client) DshbotNotificationSnapshot(ctx context.Context) (botnotify.Snapshot, error) {
	var out botnotify.Snapshot
	catalog, err := c.DshbotDescribeTyped(ctx, "")
	if err != nil {
		return out, fmt.Errorf("Bot notification catalog unavailable")
	}
	if catalog.View.NS != "dsh-bot" || catalog.View.Revision < 0 {
		return out, fmt.Errorf("Bot notification catalog is invalid")
	}
	out.Revision = catalog.View.Revision
	if len(catalog.View.Value.Items) > botnotify.MaxBots {
		return out, fmt.Errorf("Bot 通知不可用：目录超过 200 个条目 / Notification catalog exceeds the Host limit of 200 items")
	}
	profiles := map[string]DshbotCatalogItem{}
	ids := make([]string, 0, len(catalog.View.Value.Items))
	for _, p := range catalog.View.Value.Items {
		if _, exists := profiles[p.ID]; exists || p.ID == "" || p.SessionID == "" || p.Notify == nil {
			return out, fmt.Errorf("Bot notification profile identity or notify setting unavailable")
		}
		profiles[p.ID] = p
		ids = append(ids, p.ID)
	}
	if len(ids) == 0 {
		return out, nil
	}
	activity, err := c.DshbotActivityTyped(ctx, ids)
	if err != nil {
		return out, fmt.Errorf("Bot notification activity unavailable")
	}
	if len(activity.Entries) != len(ids) {
		return out, fmt.Errorf("Bot notification activity is incomplete")
	}
	seen := map[string]bool{}
	out.Revision = catalog.View.Revision
	for _, a := range activity.Entries {
		p, ok := profiles[a.BotID]
		if !ok || seen[a.BotID] || a.SessionID != p.SessionID {
			return botnotify.Snapshot{}, fmt.Errorf("Bot notification activity identity or status unavailable")
		}
		seen[a.BotID] = true
		// A cold/uninspectable Bot cannot erase another Bot's valid activity.
		// Omit this row so its old click/baseline is retired; recovery begins
		// with a new baseline rather than manufacturing an attention edge.
		if a.Error != "" {
			out.Unavailable++
			continue
		}
		if !a.NotificationShapeKnown || a.ActivitySeq < 0 || len(a.Goal) == 0 || len(a.LastRun) == 0 {
			return botnotify.Snapshot{}, fmt.Errorf("Bot notification activity status unavailable")
		}
		var goal *struct {
			Phase string `json:"phase"`
		}
		var run *struct {
			RoutineID string `json:"routineId"`
			RunID     string `json:"runId"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			Silent    *bool  `json:"silent"`
			EndedAt   int64  `json:"endedAt"`
		}
		if json.Unmarshal(a.Goal, &goal) != nil || json.Unmarshal(a.LastRun, &run) != nil {
			return botnotify.Snapshot{}, fmt.Errorf("Bot notification activity status is invalid")
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = p.ID
		}
		if p.Kind != "room" && strings.TrimSpace(p.Title) != "" {
			name = strings.TrimSpace(p.Title)
		}
		row := botnotify.Row{ID: p.ID, SessionID: p.SessionID, Name: name, Notify: *p.Notify, Hidden: p.Hidden, Kind: p.Kind, Attention: a.Attention, ActivitySeq: a.ActivitySeq}
		if goal != nil {
			row.GoalPhase = goal.Phase
		}
		if run != nil {
			if run.RoutineID == "" || run.RunID == "" || run.Status == "" || run.Silent == nil || run.EndedAt < 0 {
				return botnotify.Snapshot{}, fmt.Errorf("Bot notification run identity is invalid")
			}
			row.Run = botnotify.Run{RoutineID: run.RoutineID, RunID: run.RunID, Name: run.Name, Status: run.Status, Silent: *run.Silent, EndedAt: run.EndedAt}
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// Preserve the existing activity DTO for roster consumers while requiring the
// original notification fields to be present for this stricter observer.
func (a *DshbotActivityEntry) UnmarshalJSON(raw []byte) error {
	type plain DshbotActivityEntry
	var value plain
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	_, attention := fields["attention"]
	_, activitySeq := fields["activitySeq"]
	value.NotificationShapeKnown = attention && activitySeq && string(fields["attention"]) != "null" && string(fields["activitySeq"]) != "null" && len(value.Goal) > 0 && len(value.LastRun) > 0
	*a = DshbotActivityEntry(value)
	return nil
}

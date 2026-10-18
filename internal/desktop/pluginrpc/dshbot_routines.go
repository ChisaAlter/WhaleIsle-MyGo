package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Contract: WhaleIsle@e9f584be vendor/dshbot/lib/control-plane.js:346-430,
// 638-695. Routines live in describe.view.value, not the generic Scheduler.
// bot/trigger-token lives in a separate explicitly confirmed get-or-create helper;
// ordinary describe and routine controls never request it.
type DshbotRoutine struct {
	ID              string             `json:"id"`
	BotID           string             `json:"botId"`
	Name            string             `json:"name"`
	Prompt          string             `json:"prompt"`
	Schedule        string             `json:"schedule"`
	Timezone        string             `json:"timezone"`
	IntervalMinutes int                `json:"intervalMinutes"`
	MaxRuns         int                `json:"maxRuns"`
	RunCount        int                `json:"runCount"`
	Enabled         bool               `json:"enabled"`
	NextRunAt       int64              `json:"nextRunAt"`
	LastRunAt       int64              `json:"lastRunAt"`
	LastOutcome     string             `json:"lastOutcome"`
	LastError       string             `json:"lastError"`
	LastOutput      string             `json:"lastOutput"`
	PendingRunID    string             `json:"pendingRunId"`
	RunSessionID    string             `json:"runSessionId"`
	RunTurn         int                `json:"runTurn"`
	FailureCount    int                `json:"failureCount"`
	ContextFrom     []string           `json:"contextFrom"`
	SilentAllowed   bool               `json:"silentAllowed"`
	Watch           json.RawMessage    `json:"watch"`
	RunHistory      []DshbotRoutineRun `json:"runHistory"`
	UpdatedAt       int64              `json:"updatedAt"`
}
type DshbotRoutineRun struct {
	RunID     string `json:"runId"`
	Status    string `json:"status"`
	Trigger   string `json:"trigger"`
	SessionID string `json:"sessionId"`
	Turn      int    `json:"turn"`
	Error     string `json:"error"`
	CreatedAt int64  `json:"createdAt"`
	StartedAt int64  `json:"startedAt"`
	EndedAt   int64  `json:"endedAt"`
}
type DshbotRoutineView struct {
	NS       string `json:"ns"`
	Revision int    `json:"revision"`
	Value    struct {
		Items    []DshbotCatalogItem `json:"items"`
		Routines []DshbotRoutine     `json:"routines"`
	} `json:"value"`
}

func (v *DshbotRoutineView) Bot(id string) (DshbotCatalogItem, bool) {
	if v != nil && v.NS == "dsh-bot" && v.Revision >= 0 {
		for _, b := range v.Value.Items {
			if b.ID == id && b.Kind != "room" && !b.Hidden && strings.TrimSpace(b.SessionID) != "" {
				return b, true
			}
		}
	}
	return DshbotCatalogItem{}, false
}
func (v *DshbotRoutineView) Routine(id, botID string) (DshbotRoutine, bool) {
	if v != nil {
		for _, r := range v.Value.Routines {
			if r.ID == id && r.BotID == botID {
				return r, true
			}
		}
	}
	return DshbotRoutine{}, false
}
func (c *Client) DshbotRoutines(ctx context.Context, botID string) (*DshbotRoutineView, error) {
	if strings.TrimSpace(botID) == "" {
		return nil, fmt.Errorf("select a Bot first")
	}
	var result struct {
		View *DshbotRoutineView `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": botID}, &result); err != nil {
		return nil, err
	}
	if _, ok := result.View.Bot(botID); !ok {
		return nil, fmt.Errorf("select an available individual Bot; rooms do not own routines")
	}
	return result.View, nil
}
func (c *Client) DshbotMissedRoutines(ctx context.Context) ([]string, error) {
	var result struct {
		IDs []string `json:"ids"`
	}
	err := c.Call(ctx, "dshbot", "routine/missed", map[string]any{}, &result)
	return result.IDs, err
}

// Base fields are edited together. Advanced fields have independent presence
// flags: untouched contextFrom/watch/silentAllowed and runtime fields are omitted.
type DshbotRoutineEdit struct {
	Name, Prompt, Schedule, Timezone string
	MaxRuns                          int
	Enabled                          bool
}
type DshbotRoutineCommand struct {
	Action, ID, BotID string
	Revision          int
	EnabledBefore     bool
	Edit              DshbotRoutineEdit
	Advanced          DshbotRoutineAdvancedPatch
}

var routineDurationPattern = regexp.MustCompile(`(?i)^(every\s+)?(\d+)([mhd])$`)

func normalizedRoutineSchedule(value string) string {
	value = strings.TrimSpace(value)
	if m := routineDurationPattern.FindStringSubmatch(value); m != nil {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return value
		}
		prefix := ""
		if m[1] != "" {
			prefix = "every "
		}
		return prefix + strconv.Itoa(n) + strings.ToLower(m[3])
	}
	return value
}
func ValidateDshbotRoutineEdit(e DshbotRoutineEdit) error {
	for _, f := range []struct {
		name, value string
		limit       int
	}{{"name", e.Name, 120}, {"prompt", e.Prompt, 8000}} {
		if strings.TrimSpace(f.value) == "" || len(utf16.Encode([]rune(f.value))) > f.limit {
			return fmt.Errorf("%s is required and must be at most %d UTF-16 units", f.name, f.limit)
		}
	}
	if strings.TrimSpace(e.Schedule) == "" || strings.TrimSpace(e.Timezone) == "" {
		return fmt.Errorf("schedule and timezone are required; Host validates the original schedule syntax")
	}
	if len(utf16.Encode([]rune(strings.TrimSpace(e.Schedule)))) > 200 || strings.ContainsRune(e.Schedule, '\x00') {
		return fmt.Errorf("schedule must be at most 200 UTF-16 units and contain no NUL")
	}
	if e.MaxRuns < 0 || e.MaxRuns > 100000 {
		return fmt.Errorf("run limit must be 0–100000 (0 means unlimited)")
	}
	return nil
}
func (c *Client) DshbotControlRoutine(ctx context.Context, cmd DshbotRoutineCommand) (*DshbotRoutineView, error) {
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.BotID) == "" || cmd.Revision < 0 {
		return nil, fmt.Errorf("refresh and select an existing routine first")
	}
	payload := map[string]any{"id": cmd.ID, "revision": cmd.Revision}
	switch cmd.Action {
	case "save":
		if err := ValidateDshbotRoutineEdit(cmd.Edit); err != nil {
			return nil, err
		}
		if err := ValidateDshbotRoutineAdvanced(cmd.Advanced); err != nil {
			return nil, err
		}
		applyRoutineAdvancedPayload(payload, cmd.Advanced)
		payload["botId"] = cmd.BotID
		payload["name"] = strings.TrimSpace(cmd.Edit.Name)
		payload["prompt"] = strings.TrimSpace(cmd.Edit.Prompt)
		payload["schedule"] = strings.TrimSpace(cmd.Edit.Schedule)
		payload["timezone"] = strings.TrimSpace(cmd.Edit.Timezone)
		payload["maxRuns"] = cmd.Edit.MaxRuns
		payload["enabled"] = cmd.Edit.Enabled
	case "toggle", "run", "delete":
	default:
		return nil, fmt.Errorf("unsupported routine operation")
	}
	var result struct {
		View *DshbotRoutineView `json:"view"`
	}
	// No automatic retry: run and toggle are not idempotent.
	if err := c.Call(ctx, "dshbot", "routine/"+cmd.Action, payload, &result); err != nil {
		return nil, err
	}
	if err := ValidateDshbotRoutineReceipt(cmd, result.View); err != nil {
		return nil, err
	}
	return result.View, nil
}
func ValidateDshbotRoutineReceipt(cmd DshbotRoutineCommand, v *DshbotRoutineView) error {
	if _, ok := v.Bot(cmd.BotID); !ok || v.Revision < cmd.Revision {
		return fmt.Errorf("Host returned no matching Bot catalog; refresh before continuing")
	}
	if cmd.Action != "run" && v.Revision <= cmd.Revision {
		return fmt.Errorf("Host did not return a newer catalog revision")
	}
	if cmd.Action == "delete" {
		for _, r := range v.Value.Routines {
			if r.ID == cmd.ID {
				return fmt.Errorf("Host did not confirm routine deletion")
			}
		}
		return nil
	}
	r, ok := v.Routine(cmd.ID, cmd.BotID)
	if !ok {
		return fmt.Errorf("Host returned no matching routine; refresh before continuing")
	}
	if cmd.Action == "toggle" && r.Enabled == cmd.EnabledBefore {
		return fmt.Errorf("Host did not confirm the requested enabled state")
	}
	if cmd.Action == "save" && (r.Name != strings.TrimSpace(cmd.Edit.Name) || r.Prompt != strings.TrimSpace(cmd.Edit.Prompt) || r.Enabled != cmd.Edit.Enabled || r.MaxRuns != cmd.Edit.MaxRuns || r.Schedule != normalizedRoutineSchedule(cmd.Edit.Schedule) || r.Timezone != strings.TrimSpace(cmd.Edit.Timezone)) {
		return fmt.Errorf("Host returned different saved fields; refresh before continuing")
	}
	if cmd.Action == "save" {
		return validateRoutineAdvancedReceipt(cmd.Advanced, r)
	}
	return nil
}

package pluginrpc

import (
	"context"
	"fmt"
	"strings"
)

// The original routine/save creates only when id is absent. There is no Host
// requestId/idempotency contract; RequestID and BeforeIDs are local guards only.
type DshbotRoutineCreateCommand struct {
	BotID, SessionID, RequestID string
	Revision                    int
	Edit                        DshbotRoutineEdit
	BeforeIDs                   []string
}
type DshbotRoutineCreateResult struct {
	Rejected bool
	ID       string
	View     *DshbotRoutineView
}

func (c *Client) DshbotCreateRoutine(ctx context.Context, cmd DshbotRoutineCreateCommand) (DshbotRoutineCreateResult, error) {
	var out DshbotRoutineCreateResult
	if cmd.BotID == "" || cmd.SessionID == "" || cmd.RequestID == "" || cmd.Revision < 0 {
		out.Rejected = true
		return out, fmt.Errorf("refresh and review the selected Bot before creating")
	}
	if err := ValidateDshbotRoutineEdit(cmd.Edit); err != nil {
		out.Rejected = true
		return out, err
	}
	payload := map[string]any{"botId": cmd.BotID, "revision": cmd.Revision, "name": strings.TrimSpace(cmd.Edit.Name), "prompt": strings.TrimSpace(cmd.Edit.Prompt), "schedule": strings.TrimSpace(cmd.Edit.Schedule), "timezone": strings.TrimSpace(cmd.Edit.Timezone), "maxRuns": cmd.Edit.MaxRuns, "enabled": cmd.Edit.Enabled, "contextFrom": []string{}, "silentAllowed": true, "watch": nil}
	var receipt struct {
		View *DshbotRoutineView `json:"view"`
	}
	// Exactly one mutation. Do not retry after a lost response or call routine/run.
	if err := c.Call(ctx, "dshbot", "routine/save", payload, &receipt); err != nil {
		out.Rejected = routineCreatePreCommitRejection(err)
		return out, err
	}
	out.View = receipt.View
	id, err := DshbotRoutineCreatedID(cmd, receipt.View)
	out.ID = id
	return out, err
}
func DshbotRoutineCreatedID(cmd DshbotRoutineCreateCommand, v *DshbotRoutineView) (string, error) {
	b, ok := v.Bot(cmd.BotID)
	if !ok || b.SessionID != cmd.SessionID || v.Revision <= cmd.Revision {
		return "", fmt.Errorf("creation returned no matching newer Bot catalog; refresh to verify")
	}
	old := map[string]bool{}
	for _, id := range cmd.BeforeIDs {
		old[id] = true
	}
	found := ""
	for _, r := range v.Value.Routines {
		if old[r.ID] || r.ID == "" || r.BotID != cmd.BotID {
			continue
		}
		e := cmd.Edit
		if r.Name != strings.TrimSpace(e.Name) || r.Prompt != strings.TrimSpace(e.Prompt) || r.Schedule != normalizedRoutineSchedule(e.Schedule) || r.Timezone != strings.TrimSpace(e.Timezone) || r.MaxRuns != e.MaxRuns || r.Enabled != e.Enabled || !r.SilentAllowed || len(r.ContextFrom) > 0 || (len(r.Watch) > 0 && strings.TrimSpace(string(r.Watch)) != "null") {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("multiple new matching routines; refresh and inspect instead of creating again")
		}
		found = r.ID
	}
	if found == "" {
		return "", fmt.Errorf("no unique new routine in the receipt; refresh before another action")
	}
	return found, nil
}

// These exact messages are emitted before any write by checkRevision, bot(),
// schedule and field validation in the pinned original routine/save code.
// Other server, transport or receipt errors remain ambiguous.
func routineCreatePreCommitRejection(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	if !strings.HasPrefix(message, "pluginrpc: ") || strings.HasPrefix(message, "pluginrpc: HTTP ") {
		return false
	}
	// schedule.js wraps cron-parser failures before routine/save reaches scope.set.
	// Parse the response envelope's message boundary, not arbitrary substring matches.
	if _, hostMessage, ok := strings.Cut(strings.TrimPrefix(message, "pluginrpc: "), ": "); ok && strings.HasPrefix(hostMessage, "Invalid cron schedule: ") && len(hostMessage) > len("Invalid cron schedule: ") {
		return true
	}
	for _, known := range []string{"Bot catalog changed. Refresh and try again.", "Bot catalog is read-only.", "Routine limit reached.", "Bot is unavailable.", "Invalid routine timezone.", "Invalid name.", "Invalid prompt.", "Invalid run limit.", "Invalid routine schedule.", "Routine interval is out of range.", "Advanced schedules must use a five-field cron expression."} {
		if strings.HasSuffix(message, ": "+known) {
			return true
		}
	}
	return false
}

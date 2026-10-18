package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type DshbotObjectiveBot struct {
	DshbotCatalogItem
	Objective          string `json:"objective"`
	ObjectiveMaxRounds int    `json:"objectiveMaxRounds"`
}
type DshbotObjectiveBlocker struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// bot/activity exposes a summary only: no goal ID/revision is present here.
type DshbotObjectiveGoal struct {
	Phase         string                  `json:"phase"`
	Activation    string                  `json:"activation"`
	RoundsStarted int                     `json:"roundsStarted"`
	MaxGoalRounds int                     `json:"maxGoalRounds"`
	Blocker       *DshbotObjectiveBlocker `json:"blocker"`
}
type DshbotObjectiveStatus struct {
	Bot           DshbotObjectiveBot
	Revision      int
	Goal          *DshbotObjectiveGoal
	GoalKnown     bool
	ActivityError string
}
type dshbotObjectiveView struct {
	NS       string `json:"ns"`
	Revision int    `json:"revision"`
	Value    struct {
		Items []DshbotObjectiveBot `json:"items"`
	} `json:"value"`
}

func (v dshbotObjectiveView) status(id string) (DshbotObjectiveStatus, error) {
	out := DshbotObjectiveStatus{Revision: v.Revision}
	matches := 0
	for _, b := range v.Value.Items {
		if b.ID == id {
			out.Bot = b
			matches++
		}
	}
	if v.NS != "dsh-bot" || v.Revision < 0 || matches != 1 || out.Bot.Kind == "room" || out.Bot.Hidden || out.Bot.SessionID == "" || out.Bot.ObjectiveMaxRounds < 1 || out.Bot.ObjectiveMaxRounds > 200 {
		return out, fmt.Errorf("current individual Bot objective configuration unavailable")
	}
	return out, nil
}
func CloneDshbotObjective(s DshbotObjectiveStatus) DshbotObjectiveStatus {
	s.Bot.MemberBotIDs = append([]string(nil), s.Bot.MemberBotIDs...)
	if s.Goal != nil {
		g := *s.Goal
		if g.Blocker != nil {
			b := *g.Blocker
			g.Blocker = &b
		}
		s.Goal = &g
	}
	return s
}
func SameDshbotObjective(a, b DshbotObjectiveStatus) bool {
	return a.Revision == b.Revision && a.Bot.ID == b.Bot.ID && a.Bot.SessionID == b.Bot.SessionID && a.Bot.Objective == b.Bot.Objective && a.Bot.ObjectiveMaxRounds == b.Bot.ObjectiveMaxRounds && a.GoalKnown == b.GoalKnown && reflect.DeepEqual(a.Goal, b.Goal)
}
func readObjectiveGoal(raw json.RawMessage, full bool) (*DshbotObjectiveGoal, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("Host omitted objective status")
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}
	var goal DshbotObjectiveGoal
	if err := json.Unmarshal(raw, &goal); err != nil || goal.Phase == "" {
		return nil, fmt.Errorf("Host objective status is unavailable")
	}
	if full {
		var extra struct {
			BlockedReason *DshbotObjectiveBlocker `json:"blockedReason"`
		}
		if err := json.Unmarshal(raw, &extra); err != nil {
			return nil, err
		}
		goal.Blocker = extra.BlockedReason
	}
	return &goal, nil
}
func (c *Client) DshbotObjective(ctx context.Context, id string) (DshbotObjectiveStatus, error) {
	var out DshbotObjectiveStatus
	if strings.TrimSpace(id) == "" {
		return out, fmt.Errorf("select a Bot first")
	}
	var described struct {
		View dshbotObjectiveView `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": id}, &described); err != nil {
		return out, err
	}
	var err error
	out, err = described.View.status(id)
	if err != nil {
		return out, err
	}
	var activity struct {
		Entries []struct {
			BotID     string          `json:"botId"`
			SessionID string          `json:"sessionId"`
			Goal      json.RawMessage `json:"goal"`
			Error     string          `json:"error"`
		} `json:"entries"`
	}
	if err = c.Call(ctx, "dshbot", "bot/activity", map[string]any{"ids": []string{id}}, &activity); err != nil {
		return out, err
	}
	if len(activity.Entries) != 1 || activity.Entries[0].BotID != id || activity.Entries[0].SessionID != out.Bot.SessionID {
		return out, fmt.Errorf("Bot activity identity changed; refresh")
	}
	out.Goal, err = readObjectiveGoal(activity.Entries[0].Goal, false)
	out.GoalKnown = err == nil
	out.ActivityError = activity.Entries[0].Error
	return out, err
}

type DshbotObjectiveCommand struct {
	Action   string
	Expected DshbotObjectiveStatus
}
type DshbotObjectiveResult struct {
	Status    DshbotObjectiveStatus
	Warnings  []string
	Attempted bool
}

func ValidateDshbotObjectiveCommand(cmd DshbotObjectiveCommand) error {
	s := cmd.Expected
	if s.Bot.ID == "" || s.Bot.SessionID == "" || s.Bot.Kind == "room" || s.Bot.Hidden || s.Bot.ObjectiveMaxRounds < 1 || s.Bot.ObjectiveMaxRounds > 200 || s.Revision < 0 || !s.GoalKnown {
		return fmt.Errorf("refresh the current Bot objective before reviewing")
	}
	switch cmd.Action {
	case "start":
		if strings.TrimSpace(s.Bot.Objective) == "" {
			return fmt.Errorf("configure a Bot objective in the existing profile editor first")
		}
	case "pause", "clear":
	default:
		return fmt.Errorf("unsupported Bot objective action")
	}
	return nil
}

// Exactly one existing command, after the caller's current-identity preflight.
// start/pause have no Host CAS; adding a revision field would not create one.
func (c *Client) DshbotControlObjective(ctx context.Context, cmd DshbotObjectiveCommand) (DshbotObjectiveResult, error) {
	var out DshbotObjectiveResult
	if err := ValidateDshbotObjectiveCommand(cmd); err != nil {
		return out, err
	}
	payload := map[string]any{"id": cmd.Expected.Bot.ID}
	if cmd.Action == "clear" {
		payload["revision"] = cmd.Expected.Revision
	}
	var result struct {
		View     dshbotObjectiveView `json:"view"`
		Goal     json.RawMessage     `json:"goal"`
		Warnings []string            `json:"warnings"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/objective/"+cmd.Action, payload, &result); err != nil {
		return out, fmt.Errorf("objective action outcome uncertain; refresh status before any further action")
	}
	out.Warnings = append([]string(nil), result.Warnings...)
	var err error
	out.Status, err = result.View.status(cmd.Expected.Bot.ID)
	if err != nil {
		return out, err
	}
	if out.Status.Bot.SessionID != cmd.Expected.Bot.SessionID {
		return out, fmt.Errorf("canonical Bot session changed; refresh before continuing")
	}
	if cmd.Action == "clear" {
		if out.Status.Bot.Objective != "" {
			return out, fmt.Errorf("catalog objective clear not confirmed")
		}
		return out, nil
	}
	if out.Status.Bot.Objective != cmd.Expected.Bot.Objective || out.Status.Bot.ObjectiveMaxRounds != cmd.Expected.Bot.ObjectiveMaxRounds {
		return out, fmt.Errorf("objective configuration changed during the action; refresh")
	}
	out.Status.Goal, err = readObjectiveGoal(result.Goal, true)
	out.Status.GoalKnown = err == nil
	return out, err
}

// Clear's scope is the catalog objective and its current live goal. Its Host
// CAS is catalog-only; ordinary live progress must not masquerade as a catalog conflict.
func DshbotObjectivePreflightMatches(cmd DshbotObjectiveCommand, fresh DshbotObjectiveStatus) bool {
	expected := cmd.Expected
	if cmd.Action == "clear" {
		return expected.Revision == fresh.Revision && expected.Bot.ID == fresh.Bot.ID && expected.Bot.SessionID == fresh.Bot.SessionID && expected.Bot.Objective == fresh.Bot.Objective && expected.Bot.ObjectiveMaxRounds == fresh.Bot.ObjectiveMaxRounds
	}
	return SameDshbotObjective(expected, fresh)
}

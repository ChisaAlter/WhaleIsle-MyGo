package pluginrpc

import (
	"context"
	"fmt"
	"strings"
)

type WhaleTaskDraft struct {
	Goal     string   `json:"goal"`
	Files    []string `json:"files"`
	Criteria []string `json:"criteria,omitempty"`
}
type WhaleTaskInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Hash  string `json:"hash"`
	Bytes int64  `json:"bytes"`
}
type WhaleTaskResult struct {
	Summary         string   `json:"summary"`
	Evidence        []string `json:"evidence"`
	CoveredCriteria []string `json:"coveredCriteria"`
	Unmet           []string `json:"unmet"`
	Outcome         string   `json:"outcome"`
}
type WhaleTaskEffect struct {
	ID     string `json:"id"`
	Target string `json:"target"`
	State  string `json:"state"`
	Error  string `json:"error"`
}
type WhaleTask struct {
	ID         string           `json:"id"`
	Goal       string           `json:"goal"`
	State      string           `json:"state"`
	Error      string           `json:"error"`
	Revision   int              `json:"revision"`
	Generation int              `json:"generation"`
	Hold       bool             `json:"hold"`
	Criteria   []string         `json:"criteria"`
	Inputs     []WhaleTaskInput `json:"inputs"`
	Model      struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	} `json:"model"`
	Origin struct {
		Kind    string `json:"kind"`
		EventID string `json:"eventId"`
	} `json:"origin"`
	Execution *struct {
		SessionID string `json:"sessionId"`
		Requests  int    `json:"requests"`
		ReadBytes int64  `json:"readBytes"`
	} `json:"execution,omitempty"`
	Result  *WhaleTaskResult  `json:"result,omitempty"`
	Effects []WhaleTaskEffect `json:"effects"`
}

func (t WhaleTask) Active() bool {
	return t.State == "queued" || t.State == "running" || t.State == "stopping"
}
func (t WhaleTask) Delivering() bool {
	for _, effect := range t.Effects {
		if effect.State == "pending" || effect.State == "sending" {
			return true
		}
	}
	return false
}
func (t WhaleTask) CanResume() bool {
	return t.State == "cancelled" || t.State == "interrupted" || t.State == "failed" || t.State == "needs_user" || t.State == "unknown"
}

type WhaleTasks struct {
	Available bool        `json:"available"`
	Error     string      `json:"error"`
	Tasks     []WhaleTask `json:"tasks"`
}

func (c *Client) WhaleTasks(ctx context.Context) (WhaleTasks, error) {
	var out WhaleTasks
	err := c.Call(ctx, "dsh-whale", "tasks/list", map[string]any{}, &out)
	return out, err
}
func (c *Client) WhaleStartTask(ctx context.Context, draft WhaleTaskDraft) (WhaleTask, error) {
	var out WhaleTask
	if strings.TrimSpace(draft.Goal) == "" || len(draft.Files) == 0 {
		return out, fmt.Errorf("goal and input files are required")
	}
	err := c.Call(ctx, "dsh-whale", "tasks/start", draft, &out)
	return out, err
}
func (c *Client) WhaleControlTask(ctx context.Context, id, action string) (WhaleTask, error) {
	var out WhaleTask
	if id == "" || (action != "stop" && action != "resume") {
		return out, fmt.Errorf("invalid task control")
	}
	err := c.Call(ctx, "dsh-whale", "tasks/"+action, map[string]any{"taskId": id}, &out)
	return out, err
}
func (c *Client) WhaleTaskOwner(ctx context.Context, botID, actor string, revoke bool) error {
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("Feishu bot ID is required")
	}
	method := "tasks/owner"
	payload := map[string]any{"botId": botID}
	if revoke {
		method = "tasks/revoke-owner"
	} else {
		if strings.TrimSpace(actor) == "" || actor == "*" {
			return fmt.Errorf("one specific Feishu open_id is required")
		}
		payload["channel"], payload["actor"] = "feishu", actor
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.Call(ctx, "dsh-whale", method, payload, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("task authorization was not confirmed")
	}
	return nil
}

package pluginrpc

import (
	"context"
	"fmt"
	"strings"
)

// DshbotTask is a durable delegation record, not a session turn or a Job.
// Contract: WhaleIsle@90465b67 vendor/dshbot/lib/tasks.js and control-plane.js.
type DshbotTask struct {
	ID              string            `json:"id"`
	FromID          string            `json:"fromId"`
	FromName        string            `json:"fromName"`
	ToID            string            `json:"toId"`
	ToName          string            `json:"toName"`
	Task            string            `json:"task"`
	Constraints     []string          `json:"constraints"`
	SuccessCriteria string            `json:"successCriteria"`
	Status          string            `json:"status"`
	Attempts        int               `json:"attempts"`
	ResultSummary   string            `json:"resultSummary"`
	Error           string            `json:"error"`
	CreatedAt       int64             `json:"createdAt"`
	UpdatedAt       int64             `json:"updatedAt"`
	ParentTaskID    string            `json:"parentTaskId"`
	RetryOf         string            `json:"retryOf"`
	RetryRequestID  string            `json:"retryRequestId"`
	Events          []DshbotTaskEvent `json:"events"`
}

type DshbotTaskEvent struct {
	Type    string `json:"type"`
	ActorID string `json:"actorId"`
	Detail  string `json:"detail"`
	At      int64  `json:"at"`
}

// This read-only projection deliberately does not round-trip the whole catalog.
// Mutations go through the original revision-gated task endpoints only.
type DshbotTaskView struct {
	NS       string `json:"ns"`
	Revision int    `json:"revision"`
	Value    struct {
		Items []DshbotCatalogItem `json:"items"`
		Tasks []DshbotTask        `json:"tasks"`
	} `json:"value"`
}

func (v *DshbotTaskView) Bot(id string) (DshbotCatalogItem, bool) {
	if v != nil && v.NS == "dsh-bot" && v.Revision >= 0 {
		for _, item := range v.Value.Items {
			if item.ID == id && strings.TrimSpace(item.SessionID) != "" && !item.Hidden {
				return item, true
			}
		}
	}
	return DshbotCatalogItem{}, false
}

func DshbotTaskActionAllowed(action, status string) bool {
	switch action {
	case "pause":
		return status == "queued"
	case "resume":
		return status == "paused"
	case "cancel":
		return status == "queued" || status == "paused" || status == "delivered"
	case "retry":
		return status == "failed" || status == "cancelled"
	}
	return false
}

func (c *Client) DshbotTasks(ctx context.Context, botID string) (*DshbotTaskView, error) {
	if strings.TrimSpace(botID) == "" {
		return nil, fmt.Errorf("select a Bot first")
	}
	var result struct {
		View *DshbotTaskView `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "describe", map[string]any{"botId": botID}, &result); err != nil {
		return nil, err
	}
	if _, ok := result.View.Bot(botID); !ok {
		return nil, fmt.Errorf("Bot is unavailable in the current catalog; refresh the roster")
	}
	return result.View, nil
}

type DshbotTaskCommand struct {
	Action    string
	TaskID    string
	BotID     string
	Revision  int
	RequestID string
}

type DshbotTaskResult struct {
	View           *DshbotTaskView `json:"view"`
	TaskID         string          `json:"taskId"`
	OriginalTaskID string          `json:"originalTaskId"`
	WakeAccepted   bool            `json:"wakeAccepted"`
	WakeError      string          `json:"wakeError"`
	StopAccepted   bool            `json:"stopAccepted"`
	Created        bool            `json:"created"`
	Idempotent     bool            `json:"idempotent"`
}

// DshbotControlTask never retries transport errors. A user-requested retry must
// reuse the same RequestID until an authoritative replacement task is received.
func (c *Client) DshbotControlTask(ctx context.Context, command DshbotTaskCommand) (DshbotTaskResult, error) {
	var out DshbotTaskResult
	if command.Revision < 0 || strings.TrimSpace(command.BotID) == "" {
		return out, fmt.Errorf("refresh the Bot catalog first")
	}
	method := "task/" + command.Action
	payload := map[string]any{"revision": command.Revision}
	switch command.Action {
	case "stop":
		method = "bot/stop"
		payload["botId"] = command.BotID
	case "pause", "resume", "cancel", "retry":
		if strings.TrimSpace(command.TaskID) == "" {
			return out, fmt.Errorf("select a task first")
		}
		payload["taskId"] = command.TaskID
		if command.Action == "retry" {
			if strings.TrimSpace(command.RequestID) == "" || len(command.RequestID) > 200 {
				return out, fmt.Errorf("retry request id is required")
			}
			payload["requestId"] = command.RequestID
		}
	default:
		return out, fmt.Errorf("unsupported Bot task action")
	}
	if err := c.Call(ctx, "dshbot", method, payload, &out); err != nil {
		return out, err
	}
	if _, ok := out.View.Bot(command.BotID); !ok {
		return out, fmt.Errorf("Bot changed or the Host returned no catalog; refresh before continuing")
	}
	if command.Action == "stop" {
		if !out.StopAccepted {
			return out, fmt.Errorf("Host returned no Bot stop acceptance")
		}
		return out, nil
	}
	if out.TaskID == "" || (command.Action != "retry" && out.TaskID != command.TaskID) {
		return out, fmt.Errorf("Host returned a mismatched task; refresh before continuing")
	}
	for _, task := range out.View.Value.Tasks {
		if task.ID != out.TaskID {
			continue
		}
		if task.FromID != command.BotID && task.ToID != command.BotID {
			break
		}
		if command.Action == "retry" && (out.OriginalTaskID != command.TaskID || task.ID == command.TaskID ||
			(task.ParentTaskID != command.TaskID && task.RetryOf != command.TaskID) || task.RetryRequestID != command.RequestID) {
			break
		}
		return out, nil
	}
	return out, fmt.Errorf("Host returned no matching task receipt; refresh and verify before retrying")
}

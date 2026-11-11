// Package proactive owns the desktop's durable task inbox, approvals and timers.
// Its journal uses the event vocabulary of Whale Isle's main/proactive service.
package proactive

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("proactive item not found")
	ErrConflict      = errors.New("proactive state changed")
	ErrHumanDecision = errors.New("approval decisions require the native user interface")
)

type Task struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Autonomy       int               `json:"autonomy"`
	State          string            `json:"state"`
	Source         string            `json:"source"`
	SourceURL      string            `json:"sourceUrl,omitempty"`
	Kind           string            `json:"kind,omitempty"`
	Labels         []string          `json:"labels,omitempty"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
	ApprovalID     string            `json:"approvalId,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	LastReason     string            `json:"lastReason,omitempty"`
	LastEvidence   []json.RawMessage `json:"lastEvidence,omitempty"`
	ParentID       string            `json:"parentId,omitempty"`
	Execution      *Execution        `json:"execution,omitempty"`
	SourceItem     *SourceItem       `json:"sourceItem,omitempty"`
}

func (t Task) Terminal() bool {
	return t.State == "delivered" || t.State == "failed" || t.State == "cancelled"
}

type CreateInput struct {
	ID             string   `json:"id,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Autonomy       int      `json:"autonomy,omitempty"`
	Source         string   `json:"source,omitempty"`
	SourceURL      string   `json:"sourceUrl,omitempty"`
	Kind           string   `json:"kind,omitempty"`
	Labels         []string `json:"labels,omitempty"`
	IdempotencyKey string   `json:"idempotencyKey,omitempty"`
}

type TransitionInput struct {
	TaskID         string            `json:"taskId"`
	State          string            `json:"state"`
	Actor          string            `json:"actor,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	Evidence       []json.RawMessage `json:"evidence,omitempty"`
}

type ApprovalInput struct {
	ID        string          `json:"id,omitempty"`
	TaskID    string          `json:"taskId"`
	Action    string          `json:"action"`
	Target    json.RawMessage `json:"target,omitempty"`
	DataClass string          `json:"dataClass,omitempty"`
	Risk      string          `json:"risk,omitempty"`
}

type Approval struct {
	ID          string          `json:"id"`
	TaskID      string          `json:"taskId"`
	Action      string          `json:"action"`
	Target      json.RawMessage `json:"target,omitempty"`
	DataClass   string          `json:"dataClass"`
	Risk        string          `json:"risk"`
	Status      string          `json:"status"`
	CreatedAt   string          `json:"createdAt"`
	DecidedAt   string          `json:"decidedAt,omitempty"`
	DecidedBy   string          `json:"decidedBy,omitempty"`
	Decision    string          `json:"decision,omitempty"`
	Note        string          `json:"note,omitempty"`
	ToolCall    *ToolCall       `json:"toolCall,omitempty"`
	Effect      string          `json:"effect,omitempty"`
	EffectError string          `json:"effectError,omitempty"`
	Delivery    *Delivery       `json:"delivery,omitempty"`
}

type ScheduleInput struct {
	ID         string          `json:"id,omitempty"`
	TaskID     string          `json:"taskId"`
	At         string          `json:"at"`
	IntervalMS *int64          `json:"intervalMs,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type Job struct {
	ID         string          `json:"id"`
	TaskID     string          `json:"taskId"`
	At         string          `json:"at"`
	IntervalMS *int64          `json:"intervalMs"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	Status     string          `json:"status"`
	Execution  *RunSpec        `json:"execution,omitempty"`
}

// RunSpec is captured by a native user action. The Host control endpoint cannot
// supply it, approve it, or turn an ordinary schedule into executable work.
type RunSpec struct {
	HostHome    string `json:"hostHome"`
	Cwd         string `json:"cwd"`
	WorkspaceID string `json:"workspaceId,omitempty"`
}

type Execution struct {
	SessionID  string  `json:"sessionId"`
	RequestID  string  `json:"requestId"`
	Spec       RunSpec `json:"spec"`
	Phase      string  `json:"phase"`
	Prompt     string  `json:"prompt"`
	JobID      string  `json:"jobId,omitempty"`
	Summary    string  `json:"summary,omitempty"`
	Error      string  `json:"error,omitempty"`
	StartedAt  string  `json:"startedAt"`
	FinishedAt string  `json:"finishedAt,omitempty"`
}

func (e Execution) Unsettled() bool {
	return e.Phase == "prepared" || e.Phase == "sending" || e.Phase == "accepted" || e.Phase == "unknown"
}

type Snapshot struct {
	Tasks     []Task     `json:"tasks"`
	Outbox    []Approval `json:"outbox"`
	Schedules []Job      `json:"schedules"`
	Sources   []Source   `json:"sources"`
	Notices   []Notice   `json:"notices"`
}

type Event struct {
	Type           string            `json:"type"`
	EventID        string            `json:"eventId"`
	At             string            `json:"at"`
	Sequence       uint64            `json:"sequence"`
	Task           *Task             `json:"task,omitempty"`
	TaskID         string            `json:"taskId,omitempty"`
	State          string            `json:"state,omitempty"`
	Actor          string            `json:"actor,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	Evidence       []json.RawMessage `json:"evidence,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	Approval       *Approval         `json:"approval,omitempty"`
	ApprovalID     string            `json:"approvalId,omitempty"`
	Decision       string            `json:"decision,omitempty"`
	Note           string            `json:"note,omitempty"`
	Status         string            `json:"status,omitempty"`
	DecidedAt      string            `json:"decidedAt,omitempty"`
	DecidedBy      string            `json:"decidedBy,omitempty"`
	Job            *Job              `json:"job,omitempty"`
	JobID          string            `json:"jobId,omitempty"`
	DueAt          string            `json:"dueAt,omitempty"`
	Item           json.RawMessage   `json:"item,omitempty"`
	Execution      *Execution        `json:"execution,omitempty"`
	Source         *Source           `json:"source,omitempty"`
	SourceItem     *SourceItem       `json:"sourceItem,omitempty"`
	Notice         *Notice           `json:"notice,omitempty"`
	Tool           *ToolOperation    `json:"tool,omitempty"`
	Delivery       *Delivery         `json:"delivery,omitempty"`
}

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func validState(state string) bool {
	switch state {
	case "discovered", "triaged", "queued", "planning", "running", "waiting_input", "waiting_approval", "resumed", "verifying", "delivered", "failed", "cancelled":
		return true
	}
	return false
}

func clone[T any](value T) T {
	// All values in this package are JSON data already validated at admission.
	raw, _ := json.Marshal(value)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}

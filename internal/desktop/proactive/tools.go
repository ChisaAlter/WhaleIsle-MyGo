package proactive

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type ToolCall struct {
	TaskID    string          `json:"taskId"`
	RequestID string          `json:"requestId"`
	SessionID string          `json:"sessionId"`
	CallID    string          `json:"callId"`
	Name      string          `json:"name"`
	Source    string          `json:"source,omitempty"`
	Arguments json.RawMessage `json:"arguments"`
	Cwd       string          `json:"cwd,omitempty"`
}

type ToolOperation struct {
	Key        string   `json:"key"`
	Call       ToolCall `json:"call"`
	Action     string   `json:"action"`
	Status     string   `json:"status"`
	ApprovalID string   `json:"approvalId,omitempty"`
	Lease      string   `json:"lease,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type ToolDecision struct {
	Status     string `json:"status"`
	ApprovalID string `json:"approvalId,omitempty"`
	Lease      string `json:"lease,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func toolKey(call ToolCall) string { return call.RequestID + "/" + call.CallID }
func sameCall(a, b ToolCall) bool {
	return a.TaskID == b.TaskID && a.RequestID == b.RequestID && a.SessionID == b.SessionID && a.CallID == b.CallID && a.Name == b.Name && a.Source == b.Source && a.Cwd == b.Cwd && sameJSON(a.Arguments, b.Arguments)
}

func (s *Service) toolTask(call ToolCall) (Task, error) {
	t, ok := s.tasks[call.TaskID]
	if !ok {
		return Task{}, ErrNotFound
	}
	if t.Terminal() || t.Execution == nil || !t.Execution.Unsettled() || t.Execution.RequestID != call.RequestID || t.Execution.SessionID != call.SessionID {
		return Task{}, fmt.Errorf("%w: tool belongs to an inactive execution", ErrConflict)
	}
	return t, nil
}

// BindToolSession validates the native-owned execution before the Host writes
// its informational Session marker. No task permissions come from that marker.
func (s *Service) BindToolSession(call ToolCall) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.toolTask(call)
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": t.ID, "requestId": t.Execution.RequestID, "sessionId": t.Execution.SessionID, "autonomy": t.Autonomy}, nil
}

func toolAction(t Task, call ToolCall) string {
	if !((call.Source == "harness:fs" && (call.Name == "read" || call.Name == "write" || call.Name == "edit")) ||
		(call.Source == "harness:fs-search" && (call.Name == "glob" || call.Name == "grep"))) {
		return "unclassified"
	}
	switch call.Name {
	case "read", "glob", "grep":
		return "read"
	case "write", "edit":
		if call.Cwd != "" && filepath.Clean(call.Cwd) != filepath.Clean(t.Execution.Spec.Cwd) {
			return "outside_scope"
		}
		var args struct {
			FilePath           string `json:"file_path"`
			SandboxPermissions string `json:"sandbox_permissions"`
		}
		if json.Unmarshal(call.Arguments, &args) != nil || args.FilePath == "" || args.SandboxPermissions == "require_escalated" {
			return "outside_scope"
		}
		path := args.FilePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(t.Execution.Spec.Cwd, path)
		}
		rel, err := filepath.Rel(t.Execution.Spec.Cwd, filepath.Clean(path))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "outside_scope"
		}
		return "local_write"
	default:
		return "unclassified"
	}
}

func (s *Service) IsToolApproval(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.approvals[id]
	return ok && a.ToolCall != nil
}

func (s *Service) settleTools(task Task) []Event {
	var events []Event
	if task.Execution == nil {
		return events
	}
	for _, stored := range s.tools {
		if stored.Call.TaskID != task.ID || stored.Call.RequestID != task.Execution.RequestID {
			continue
		}
		op := stored
		switch op.Status {
		case "pending", "ready":
			op.Status = "cancelled"
		case "executing":
			op.Status = "unknown"
			op.Error = "Execution ended before a tool result receipt was recorded"
		default:
			continue
		}
		events = append(events, Event{Type: "tool.operation", Tool: &op})
		if a, ok := s.approvals[op.ApprovalID]; ok && a.Status == "pending" {
			events = append(events, Event{Type: "outbox.updated", ApprovalID: a.ID, Status: "cancelled", DecidedAt: timestamp(s.now()), DecidedBy: "execution"})
		}
	}
	return events
}

func (s *Service) PrepareTool(call ToolCall) (ToolDecision, error) {
	if call.Name == "" || call.CallID == "" || !json.Valid(call.Arguments) {
		return ToolDecision{}, fmt.Errorf("tool identity and JSON arguments are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.toolTask(call)
	if err != nil {
		return ToolDecision{}, err
	}
	key := toolKey(call)
	if old, ok := s.tools[key]; ok {
		if !sameCall(old.Call, call) {
			return ToolDecision{}, fmt.Errorf("%w: tool call parameters changed", ErrConflict)
		}
		return s.toolDecision(old), nil
	}
	if t.ApprovalID != "" {
		return ToolDecision{Status: "busy", Reason: "Another operation is awaiting approval"}, nil
	}
	action := toolAction(t, call)
	decision, err := Evaluate(PolicyInput{Autonomy: t.Autonomy, Action: action})
	if err != nil {
		return ToolDecision{}, err
	}
	op := ToolOperation{Key: key, Call: clone(call), Action: action, Status: "ready"}
	if !decision.Allowed && !decision.RequiresApproval {
		op.Status = "denied"
		op.Error = decision.Reason
	}
	events := []Event{}
	if !decision.Allowed && decision.RequiresApproval {
		id := rand.Text()
		target, _ := json.Marshal(map[string]any{"tool": call.Name, "source": call.Source, "arguments": json.RawMessage(call.Arguments), "workingDirectory": t.Execution.Spec.Cwd})
		a := Approval{ID: id, TaskID: t.ID, Action: call.Name, Target: target, DataClass: "tool-arguments", Risk: "review-required", Status: "pending", CreatedAt: timestamp(s.now()), ToolCall: &call, Effect: "pending"}
		op.Status, op.ApprovalID = "pending", id
		events = append(events, Event{Type: "approval.requested", Approval: &a}, Event{Type: "task.transitioned", TaskID: t.ID, State: "waiting_approval", Actor: "policy", Reason: "Tool execution requires approval"})
	}
	events = append(events, Event{Type: "tool.operation", Tool: &op})
	if err := s.commit(events...); err != nil {
		return ToolDecision{}, err
	}
	return s.toolDecision(op), nil
}

func (s *Service) toolDecision(op ToolOperation) ToolDecision {
	d := ToolDecision{Status: op.Status, ApprovalID: op.ApprovalID, Reason: op.Error}
	if op.Status == "pending" {
		a := s.approvals[op.ApprovalID]
		switch a.Status {
		case "approved":
			d.Status = "ready"
		case "rejected", "cancelled":
			d.Status = "denied"
			d.Reason = "User rejected or cancelled this operation"
		}
	}
	return d
}

// ClaimTool is an at-most-once dispatch lease. Unknown results and repeated
// claims never authorize a second effect, even after Host/desktop restart.
func (s *Service) ClaimTool(call ToolCall) (ToolDecision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.toolTask(call); err != nil {
		return ToolDecision{}, err
	}
	op, ok := s.tools[toolKey(call)]
	if !ok || !sameCall(op.Call, call) {
		return ToolDecision{}, ErrConflict
	}
	decision := s.toolDecision(op)
	if decision.Status != "ready" {
		return decision, nil
	}
	op.Status, op.Lease = "executing", rand.Text()
	if err := s.commit(Event{Type: "tool.operation", Tool: &op}); err != nil {
		return ToolDecision{}, err
	}
	return ToolDecision{Status: "allow", Lease: op.Lease, ApprovalID: op.ApprovalID}, nil
}

func (s *Service) FinishTool(call ToolCall, lease, status, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.tools[toolKey(call)]
	if !ok || !sameCall(op.Call, call) || op.Lease != lease || lease == "" {
		return ErrConflict
	}
	if status != "succeeded" && status != "failed" && status != "unknown" {
		return fmt.Errorf("invalid tool effect result")
	}
	if op.Status == status && op.Error == message {
		return nil
	}
	if op.Status != "executing" {
		return ErrConflict
	}
	op.Status, op.Error = status, message
	return s.commit(Event{Type: "tool.operation", Tool: &op})
}

func (s *Service) AbandonTool(call ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.tools[toolKey(call)]
	if !ok {
		return nil
	}
	if !sameCall(op.Call, call) {
		return ErrConflict
	}
	if op.Status != "pending" && op.Status != "ready" {
		return nil
	}
	op.Status = "cancelled"
	events := []Event{{Type: "tool.operation", Tool: &op}}
	if a, ok := s.approvals[op.ApprovalID]; ok && (a.Status == "pending" || a.Status == "approved") {
		events = append(events, Event{Type: "outbox.updated", ApprovalID: a.ID, Status: "cancelled", DecidedAt: timestamp(s.now()), DecidedBy: "execution"})
		if t, ok := s.tasks[call.TaskID]; ok && !t.Terminal() && t.ApprovalID == a.ID {
			events = append(events, Event{Type: "task.transitioned", TaskID: t.ID, State: "waiting_input", Actor: "executor", Reason: "Tool operation was cancelled"})
		}
	}
	return s.commit(events...)
}

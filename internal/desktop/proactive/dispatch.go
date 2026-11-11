package proactive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Dispatch is callable by the Host control plane. User decisions are excluded;
// the native Outbox calls Decide with the exact approval displayed to the user.
func (s *Service) Dispatch(ctx context.Context, action string, payload json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if !json.Valid(payload) || payload[0] != '{' {
		return nil, fmt.Errorf("proactive payload must be an object")
	}
	switch action {
	case "delivery/claim", "delivery/finish":
		var in struct {
			ApprovalID string   `json:"approvalId"`
			HostHome   string   `json:"hostHome"`
			Lease      string   `json:"lease"`
			Status     string   `json:"status"`
			Error      string   `json:"error"`
			MessageIDs []string `json:"messageIds"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		if action == "delivery/claim" {
			return s.ClaimDelivery(in.ApprovalID, in.HostHome)
		}
		return map[string]bool{"ok": true}, s.FinishDelivery(in.ApprovalID, in.Lease, in.Status, in.Error, in.MessageIDs)
	case "tool/bind", "tool/prepare", "tool/claim", "tool/finish", "tool/abandon":
		var in struct {
			Call   ToolCall `json:"call"`
			Lease  string   `json:"lease"`
			Status string   `json:"status"`
			Error  string   `json:"error"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		switch action {
		case "tool/bind":
			return s.BindToolSession(in.Call)
		case "tool/prepare":
			return s.PrepareTool(in.Call)
		case "tool/claim":
			return s.ClaimTool(in.Call)
		case "tool/finish":
			return map[string]bool{"ok": true}, s.FinishTool(in.Call, in.Lease, in.Status, in.Error)
		default:
			return map[string]bool{"ok": true}, s.AbandonTool(in.Call)
		}
	case "snapshot":
		return s.Snapshot(), nil
	case "task/create":
		var in CreateInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return s.Create(in)
	case "task/transition":
		var in TransitionInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		task, err := s.Get(in.TaskID)
		if err != nil {
			return nil, err
		}
		if task.Execution != nil {
			return nil, fmt.Errorf("native execution state is owned by the runner and user interface")
		}
		return s.Transition(in)
	case "task/get":
		var in struct {
			TaskID string `json:"taskId"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return s.Get(in.TaskID)
	case "inbox/list":
		return s.Snapshot().Tasks, nil
	case "outbox/list":
		return s.Snapshot().Outbox, nil
	case "approval/request":
		var in ApprovalInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return s.RequestApproval(in)
	case "approval/decide":
		return nil, ErrHumanDecision
	case "schedule/create":
		var in ScheduleInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return s.Schedule(in)
	case "schedule/cancel":
		var in struct {
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		return s.CancelSchedule(in.JobID)
	case "schedule/list":
		return s.Snapshot().Schedules, nil
	case "schedule/poll":
		var in struct {
			At string `json:"at"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		var at time.Time
		if in.At != "" {
			var err error
			at, err = time.Parse(time.RFC3339Nano, in.At)
			if err != nil {
				return nil, err
			}
		}
		return s.Poll(at)
	default:
		return nil, fmt.Errorf("unknown proactive action %q", action)
	}
}

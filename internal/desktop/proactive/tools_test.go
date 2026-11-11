package proactive

import (
	"encoding/json"
	"errors"
	"testing"
)

func toolTaskFixture(t *testing.T, s *Service, dir string, level int) ToolCall {
	t.Helper()
	task, err := s.Create(CreateInput{Title: "review exact operation", Autonomy: level})
	if err != nil {
		t.Fatal(err)
	}
	task, err = s.PrepareExecution(task.ID, RunSpec{HostHome: "host", Cwd: dir}, false)
	if err != nil {
		t.Fatal(err)
	}
	return ToolCall{TaskID: task.ID, SessionID: task.Execution.SessionID, RequestID: task.Execution.RequestID, CallID: "call-1", Name: "send_external", Arguments: json.RawMessage(`{"recipient":"named-owner","text":"approved body"}`)}
}

func TestToolApprovalExactParametersAndOnceAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	call := toolTaskFixture(t, s, dir, 5)
	d, err := s.PrepareTool(call)
	if err != nil || d.Status != "pending" {
		t.Fatal(d, err)
	}
	changed := call
	changed.Arguments = json.RawMessage(`{"recipient":"other","text":"approved body"}`)
	if _, err := s.PrepareTool(changed); !errors.Is(err, ErrConflict) {
		t.Fatal("arguments changed behind approval", err)
	}
	largeID := call
	largeID.CallID = "large-id"
	largeID.Arguments = json.RawMessage(`{"recipientId":9007199254740992}`)
	// Equality must not round a recipient identifier through float64.
	if sameJSON(largeID.Arguments, json.RawMessage(`{"recipientId":9007199254740993}`)) {
		t.Fatal("large recipient IDs compared equal")
	}
	if claimed, err := s.ClaimTool(call); err != nil || claimed.Status != "pending" {
		t.Fatal("unapproved tool dispatched", claimed, err)
	}
	if _, err := s.Decide(d.ApprovalID, "approved", ""); err != nil {
		t.Fatal(err)
	}
	s = openService(t, dir)
	claimed, err := s.ClaimTool(call)
	if err != nil || claimed.Status != "allow" || claimed.Lease == "" {
		t.Fatal(claimed, err)
	}
	if duplicate, err := s.ClaimTool(call); err != nil || duplicate.Status == "allow" {
		t.Fatal("duplicate effect authorized", duplicate, err)
	}
	s = openService(t, dir)
	if duplicate, err := s.ClaimTool(call); err != nil || duplicate.Status == "allow" {
		t.Fatal("restart repeated effect", duplicate, err)
	}
	if err := s.FinishTool(call, claimed.Lease, "succeeded", ""); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Outbox[0].Effect != "succeeded" {
		t.Fatal("missing delivery receipt")
	}
	next := call
	next.CallID = "call-2"
	d, err = s.PrepareTool(next)
	if err != nil || d.Status != "pending" {
		t.Fatal("second invocation reused approval", d, err)
	}
	if _, err := s.Decide(d.ApprovalID, "rejected", ""); err != nil {
		t.Fatal(err)
	}
	denied, err := s.ClaimTool(next)
	if err != nil || denied.Status != "denied" {
		t.Fatal(denied, err)
	}
}

func TestToolPolicyScopesAndInterruptionReceipts(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	readOnly := toolTaskFixture(t, s, dir, 1)
	if d, err := s.PrepareTool(readOnly); err != nil || d.Status != "denied" {
		t.Fatal(d, err)
	}
	readOnly.Name = "read"
	readOnly.CallID = "impersonated-read"
	readOnly.Source = "plugin:unclassified"
	readOnly.Arguments = json.RawMessage(`{"file_path":"note.txt"}`)
	if d, err := s.PrepareTool(readOnly); err != nil || d.Status != "denied" {
		t.Fatal("plugin impersonating read bypassed L1 denial", d, err)
	}
	readOnly.CallID = "read"
	readOnly.Source = "harness:fs"
	if d, err := s.PrepareTool(readOnly); err != nil || d.Status != "ready" {
		t.Fatal(d, err)
	}
	search := readOnly
	search.Name = "grep"
	search.CallID = "grep"
	search.Source = "harness:fs-search"
	search.Arguments = json.RawMessage(`{"pattern":"todo"}`)
	if d, err := s.PrepareTool(search); err != nil || d.Status != "ready" {
		t.Fatal(d, err)
	}
	write := toolTaskFixture(t, s, dir, 2)
	write.Name = "write"
	write.Arguments = json.RawMessage(`{"file_path":"note.txt","content":"scoped"}`)
	impersonatedWrite := write
	impersonatedWrite.CallID = "impersonated-write"
	impersonatedWrite.Source = "plugin:unclassified"
	impDecision, err := s.PrepareTool(impersonatedWrite)
	if err != nil || impDecision.Status != "pending" {
		t.Fatal("unclassified plugin write auto-authorized", impDecision, err)
	}
	if err := s.AbandonTool(impersonatedWrite); err != nil {
		t.Fatal(err)
	}
	write.Source = "harness:fs"
	if d, err := s.PrepareTool(write); err != nil || d.Status != "ready" {
		t.Fatal(d, err)
	}
	outside := write
	outside.CallID = "outside"
	outside.Arguments = json.RawMessage(`{"file_path":"../outside.txt","content":"review"}`)
	d, err := s.PrepareTool(outside)
	if err != nil || d.Status != "pending" {
		t.Fatal("outside scope auto-authorized", d, err)
	}
	if _, err := s.updateExecution(write.TaskID, write.RequestID, "interrupted", "waiting_input", "", ""); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Outbox[0].Status != "cancelled" {
		t.Fatal("interruption left a live approval")
	}
	if _, err := s.Decide(d.ApprovalID, "approved", ""); !errors.Is(err, ErrConflict) {
		t.Fatal("late approval revived stopped execution", err)
	}
}

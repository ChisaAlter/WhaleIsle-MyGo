package proactive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func openService(t *testing.T, dir string) *Service {
	t.Helper()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func createTask(t *testing.T, s *Service, title string) Task {
	t.Helper()
	task, err := s.Create(CreateInput{Title: title})
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestJournalRestartApprovalsAndSchedule(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	input := CreateInput{Title: "核对资料", Description: strings.Repeat("长内容", 24000), Source: "github", SourceURL: "https://example.invalid/issue/1", Autonomy: 3, IdempotencyKey: "discovery-1"}
	task, err := s.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate, err := s.Create(input); err != nil || duplicate.ID != task.ID {
		t.Fatal(duplicate, err)
	}
	at := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	job, err := s.Schedule(ScheduleInput{TaskID: task.ID, At: timestamp(at)})
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.RequestApproval(ApprovalInput{TaskID: task.ID, Action: "发送核对结果", Target: json.RawMessage(`{"channel":"feishu","recipient":"named-owner"}`), Risk: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Dispatch(context.Background(), "approval/decide", json.RawMessage(`{"approvalId":"`+a.ID+`","decision":"approved","actor":"user"}`)); !errors.Is(err, ErrHumanDecision) {
		t.Fatalf("control token approved own request: %v", err)
	}
	if _, err := s.Transition(TransitionInput{TaskID: task.ID, State: "running"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("pending approval bypassed: %v", err)
	}
	if due, err := s.Poll(at); err != nil || len(due) != 0 {
		t.Fatalf("approval consumed by timer: %+v %v", due, err)
	}
	s = openService(t, dir)
	restored, _ := s.Get(task.ID)
	if restored.Description != input.Description || restored.State != "waiting_approval" || restored.ApprovalID != a.ID {
		t.Fatal("task did not replay losslessly")
	}
	if _, err := s.Decide(a.ID, "approved", "只发送到指定收件人"); err != nil {
		t.Fatal(err)
	}
	due, err := s.Poll(at)
	if err != nil || len(due) != 1 || due[0].ID != job.ID {
		t.Fatal(due, err)
	}
	s = openService(t, dir)
	if due, err := s.Poll(at.Add(time.Hour)); err != nil || len(due) != 0 {
		t.Fatalf("one-shot replayed: %v %v", due, err)
	}
	got := s.Snapshot()
	if got.Tasks[0].State != "resumed" || got.Tasks[0].ApprovalID != "" || got.Outbox[0].Note != "只发送到指定收件人" || got.Outbox[0].Status != "approved" {
		t.Fatalf("decision did not replay: %+v", got)
	}
	if _, err := s.Decide(a.ID, "rejected", ""); !errors.Is(err, ErrConflict) {
		t.Fatal("decision was overwritten", err)
	}
}

func TestTerminalTaskCancelsApprovalAndRecurringSchedule(t *testing.T) {
	s := openService(t, t.TempDir())
	task := createTask(t, s, "停止跟进")
	interval := int64(60_000)
	at := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	j, err := s.Schedule(ScheduleInput{TaskID: task.ID, At: timestamp(at), IntervalMS: &interval})
	if err != nil {
		t.Fatal(err)
	}
	if due, err := s.Poll(at); err != nil || len(due) != 1 {
		t.Fatal(due, err)
	}
	if s.Snapshot().Schedules[0].At != timestamp(at.Add(time.Minute)) {
		t.Fatal("recurrence drifted")
	}
	a, err := s.RequestApproval(ApprovalInput{TaskID: task.ID, Action: "发布报告"})
	if err != nil {
		t.Fatal(err)
	}
	in := TransitionInput{TaskID: task.ID, State: "cancelled", IdempotencyKey: "cancel-1"}
	if _, err := s.Transition(in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transition(in); err != nil {
		t.Fatal("replayed cancellation failed", err)
	}
	if _, err := s.Decide(a.ID, "approved", ""); !errors.Is(err, ErrConflict) {
		t.Fatal("cancelled approval revived task", err)
	}
	if due, err := s.Poll(at.Add(3 * time.Minute)); err != nil || len(due) != 0 {
		t.Fatal(due, err)
	}
	if job, err := s.CancelSchedule(j.ID); err != nil || job.Status != "cancelled" {
		t.Fatal(job, err)
	}
	if s.Snapshot().Outbox[0].Status != "cancelled" {
		t.Fatal("stale approval stayed actionable")
	}
}

func TestJournalConflictAndCorruptionPreserveOriginal(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := createTask(t, s, "原始任务")
	stale := openService(t, dir)
	createTask(t, s, "另一次写入")
	before, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stale.Create(CreateInput{Title: "不得覆盖"}); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.path)
	if !bytes.Equal(before, after) || len(stale.Snapshot().Tasks) != 1 {
		t.Fatal("failed write changed disk or memory")
	}
	corrupt := append(bytes.Clone(before), []byte(`{"type":"approval.requested"}`)...)
	if err := os.WriteFile(s.path, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("invalid record silently ignored")
	}
	after, _ = os.ReadFile(s.path)
	if !bytes.Equal(corrupt, after) {
		t.Fatal("corrupt original overwritten")
	}
	if existing, _ := stale.Get(task.ID); existing.Title != "原始任务" {
		t.Fatal(existing)
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(s.path), ".proactive-*.tmp"))
	if len(left) != 0 {
		t.Fatal("failed transaction left temporary files", left)
	}
}

func TestJournalSerializesConcurrentTasksAndKeepsUnknownEvents(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	createTask(t, s, "起点")
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("{\"type\":\"future.event\",\"unknownField\":{\"keep\":true}}\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	s = openService(t, dir)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Go(func() {
			if _, err := s.Create(CreateInput{Title: "并发提案"}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	restored := openService(t, dir)
	if len(restored.Snapshot().Tasks) != 13 {
		t.Fatal("lost concurrent task")
	}
	raw, _ := os.ReadFile(s.path)
	if !bytes.Contains(raw, []byte(`"unknownField":{"keep":true}`)) {
		t.Fatal("future data discarded")
	}
	copy := restored.Snapshot()
	copy.Tasks[0].Title = "修改副本"
	if reflect.DeepEqual(copy, restored.Snapshot()) {
		t.Fatal("snapshot aliases service")
	}
}

func TestAutonomyPolicyPreservesExternalApproval(t *testing.T) {
	for level := 1; level <= 5; level++ {
		read, err := Evaluate(PolicyInput{Autonomy: level, Action: "read"})
		if err != nil || !read.Allowed {
			t.Fatal(read, err)
		}
		send, err := Evaluate(PolicyInput{Autonomy: level, Action: "external_message", ExplicitAuthorization: true})
		if err != nil || send.Allowed {
			t.Fatal("external action approved without decision", send, err)
		}
	}
	draft, _ := Evaluate(PolicyInput{Autonomy: 2, Action: "draft"})
	if !draft.Allowed {
		t.Fatal(draft)
	}
	unknown, _ := Evaluate(PolicyInput{Autonomy: 5, Action: "other"})
	if unknown.Allowed || !unknown.RequiresApproval {
		t.Fatal(unknown)
	}
}

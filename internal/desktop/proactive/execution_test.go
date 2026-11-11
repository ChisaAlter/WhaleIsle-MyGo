package proactive

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestExecutableScheduleAtomicClaimRecoveryAndApproval(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	template := createTask(t, s, "每日核对")
	spec := RunSpec{HostHome: "host-a", Cwd: dir}
	at := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	interval := int64(60_000)
	j, err := s.ScheduleExecution(ScheduleInput{TaskID: template.ID, At: timestamp(at), IntervalMS: &interval}, spec)
	if err != nil {
		t.Fatal(err)
	}
	if polled, err := s.Poll(at); err != nil || len(polled) != 0 {
		t.Fatal("host consumed native schedule", polled, err)
	}
	if tasks, err := s.claimDue("other-host", at); err != nil || len(tasks) != 0 {
		t.Fatal("foreign Host claimed run", tasks, err)
	}
	tasks, err := s.claimDue(spec.HostHome, at.Add(3*time.Minute+20*time.Second))
	if err != nil || len(tasks) != 1 {
		t.Fatal(tasks, err)
	}
	child := tasks[0]
	if child.ParentID != template.ID || child.Execution.JobID != j.ID || child.Execution.Phase != "prepared" {
		t.Fatal(child)
	}
	if s.Snapshot().Schedules[0].At != timestamp(at.Add(4*time.Minute)) {
		t.Fatal("missed jobs did not coalesce")
	}
	s = openService(t, dir)
	if next, err := s.claimDue(spec.HostHome, at.Add(5*time.Minute)); err != nil || len(next) != 0 {
		t.Fatal("overlapping run after restart", next, err)
	}
	if _, err := s.Transition(TransitionInput{TaskID: child.ID, State: "delivered"}); !errors.Is(err, ErrConflict) {
		t.Fatal("unsettled execution overwritten", err)
	}
	if _, err := s.updateExecution(child.ID, child.Execution.RequestID, "completed", "verifying", "result", ""); err != nil {
		t.Fatal(err)
	}
	a, err := s.RequestApproval(ApprovalInput{TaskID: child.ID, Action: "发送结果"})
	if err != nil {
		t.Fatal(err)
	}
	if next, err := s.claimDue(spec.HostHome, at.Add(5*time.Minute)); err != nil || len(next) != 0 {
		t.Fatal("pending approval overlapped", next, err)
	}
	if _, err := s.Decide(a.ID, "approved", ""); err != nil {
		t.Fatal(err)
	}
	next, err := s.claimDue(spec.HostHome, at.Add(5*time.Minute))
	if err != nil || len(next) != 1 || next[0].Execution.SessionID == child.Execution.SessionID {
		t.Fatal(next, err)
	}
	if _, err := s.CancelSchedule(j.ID); err != nil {
		t.Fatal(err)
	}
	if jobs, err := s.claimDue(spec.HostHome, at.Add(time.Hour)); err != nil || len(jobs) != 0 {
		t.Fatal("cancelled job fired", jobs, err)
	}
}

func TestExecutionIdentityContinuationAndHostCannotAuthorize(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := createTask(t, s, "检查授权路径")
	spec := RunSpec{HostHome: "host-a", Cwd: dir}
	first, err := s.PrepareExecution(task.ID, spec, false)
	if err != nil {
		t.Fatal(err)
	}
	reopened := openService(t, dir)
	saved, _ := reopened.Get(task.ID)
	if saved.Execution.RequestID != first.Execution.RequestID || saved.Execution.SessionID != first.Execution.SessionID {
		t.Fatal("execution identity not durable")
	}
	if _, err := s.PrepareExecution(task.ID, spec, true); !errors.Is(err, ErrConflict) {
		t.Fatal("resubmitted unsettled execution", err)
	}
	if _, err := s.updateExecution(task.ID, first.Execution.RequestID, "interrupted", "waiting_input", "partial", ""); err != nil {
		t.Fatal(err)
	}
	foreign := spec
	foreign.HostHome = "host-b"
	if _, err := s.PrepareExecution(task.ID, foreign, true); !errors.Is(err, ErrConflict) {
		t.Fatal("continued against wrong Host", err)
	}
	second, err := s.PrepareExecution(task.ID, spec, true)
	if err != nil || second.Execution.SessionID != first.Execution.SessionID || second.Execution.RequestID == first.Execution.RequestID {
		t.Fatal(second, err)
	}
	if _, err := s.updateExecution(task.ID, first.Execution.RequestID, "completed", "verifying", "old result", ""); !errors.Is(err, ErrConflict) {
		t.Fatal("late result replaced new run", err)
	}
	transition, _ := json.Marshal(map[string]any{"taskId": task.ID, "state": "delivered", "actor": "user"})
	if _, err := s.Dispatch(context.Background(), "task/transition", transition); err == nil {
		t.Fatal("Host overwrote native execution state")
	}
	other := createTask(t, s, "Host 提案")
	payload, _ := json.Marshal(map[string]any{"taskId": other.ID, "at": "2026-11-12T08:00:00Z", "execution": spec})
	if _, err := s.Dispatch(context.Background(), "schedule/create", payload); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Schedules[0].Execution != nil {
		t.Fatal("Host supplied its own execution authorization")
	}
}

func TestTaskObservationTracksExactTurn(t *testing.T) {
	o := taskObservation{requestID: "owned"}
	for _, raw := range []string{
		`{"type":"turn/start","data":{"turn":1}}`,
		`{"type":"user/message","data":{"source":{"rpcId":"other"}}}`,
		`{"type":"assistant/message","data":{"turn":1,"message":{"content":[{"type":"text","text":"wrong"}]}}}`,
		`{"type":"turn/end","data":{"turn":1,"reason":{"kind":"completed"}}}`,
		`{"type":"turn/start","data":{"turn":2}}`,
		`{"type":"user/message","data":{"source":{"rpcId":"owned"}}}`,
		`{"type":"assistant/message","data":{"turn":2,"message":{"content":[{"type":"text","text":"owned result"}]}}}`,
		`{"type":"turn/end","data":{"turn":2,"reason":{"kind":"completed"}}}`,
	} {
		o.apply(json.RawMessage(raw))
	}
	if !o.done || o.result().Summary != "owned result" || o.result().State != "verifying" {
		t.Fatal(o)
	}
}

func TestRunnerLifecycleFreezeDoesNotAcceptNewExecution(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := createTask(t, s, "暂停期间的新任务")
	r := NewRunner(context.Background(), s, Host{}, "host-a", nil)
	defer r.Close()
	restore := r.FreezeAdmissions()
	if _, err := r.Start(task.ID, RunSpec{HostHome: "host-a", Cwd: dir}, false); err == nil {
		t.Fatal("lifecycle gate accepted task")
	}
	got, _ := s.Get(task.ID)
	if got.Execution != nil {
		t.Fatal("blocked admission wrote execution")
	}
	restore()
	restore()
	r.mu.Lock()
	frozen := r.frozen
	r.mu.Unlock()
	if frozen != 0 {
		t.Fatal("lifecycle restore not idempotent", frozen)
	}
}

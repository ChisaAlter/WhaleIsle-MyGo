package proactive

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestMonitorZeroHzIdleAndImmediateServiceWake(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)

	var calls atomic.Int32
	reader := readerFunc(func(context.Context, Source) ([]SourceItem, error) {
		calls.Add(1)
		return nil, nil
	})
	m := NewMonitor(context.Background(), s, reader, nil, nil)
	defer m.Close()

	// No enabled sources: Monitor parks cleanly with 0 reader calls.
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("unexpected reader calls while idle with no sources: %d", calls.Load())
	}

	// Configuring an enabled source wakes Monitor immediately (far faster than 1s).
	start := time.Now()
	src, err := s.ConfigureSource("owner/repo", true)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return calls.Load() == 1 })
	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("Monitor took %v to wake on ConfigureSource (expected event-driven)", elapsed)
	}

	// Once ApplyDiscovery advances NextCheck by 15m, Monitor sleeps silently again.
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("Monitor polled unexpectedly before NextCheck: %d", calls.Load())
	}

	// RefreshSource also wakes Monitor immediately through Service subscription.
	if err := s.RefreshSource(src.ID); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return calls.Load() == 2 })
	// Production owns one monitor. Retire the source-only instance before
	// exercising notification delivery, otherwise it can claim the same notice.
	m.Close()

	// RetryNotice wakes Monitor immediately to deliver pending notifications.
	task := resultTask(t, s, dir)
	id := "result:" + task.Execution.RequestID
	if _, err := s.ClaimNotice(id); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishNotice(id, "failed", "retry later"); err != nil {
		t.Fatal(err)
	}
	var retried atomic.Bool
	mRetry := NewMonitor(context.Background(), s, nil, func(context.Context, Notice) (string, error) {
		retried.Store(true)
		return "shown", nil
	}, nil)
	defer mRetry.Close()
	if err := s.RetryNotice(id); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return retried.Load() && s.Notices()[0].DesktopStatus == "shown" })
}

func TestRunnerZeroHzIdleSchedulesPromptlyAndWakesOnUnfreeze(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	r := NewRunner(context.Background(), s, Host{}, "host-a", nil)
	defer r.Close()

	// Next wait with zero scheduled jobs or ready deliveries must report disabled timer (0 Hz).
	if d, ok := r.nextWakeDelay(time.Now()); ok || d != 0 {
		t.Fatalf("expected 0 Hz wait when idle, got delay=%v ok=%v", d, ok)
	}

	// Scheduling a job 40ms in the future sets exact one-shot timer and fires without 1s polling.
	task := createTask(t, s, "按需定时触发")
	spec := RunSpec{HostHome: "host-a", Cwd: dir}
	dueAt := time.Now().Add(40 * time.Millisecond)
	if _, err := s.ScheduleExecution(ScheduleInput{TaskID: task.ID, At: timestamp(dueAt)}, spec); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		snap := s.Snapshot()
		return len(snap.Schedules) == 1 && snap.Schedules[0].Status == "fired"
	})

	// While frozen, runner parks cleanly (0 Hz); restoring admissions immediately claims due job.
	restore := r.FreezeAdmissions()
	if d, ok := r.nextWakeDelay(time.Now()); ok || d != 0 {
		t.Fatalf("expected 0 Hz wait while frozen, got delay=%v ok=%v", d, ok)
	}
	t2 := createTask(t, s, "解冻立即执行")
	if _, err := s.ScheduleExecution(ScheduleInput{TaskID: t2.ID, At: timestamp(time.Now().Add(-time.Second))}, spec); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if snap := s.Snapshot(); snap.Schedules[1].Status != "scheduled" {
		t.Fatalf("frozen runner fired job early: %+v", snap.Schedules[1])
	}
	start := time.Now()
	restore()
	waitFor(t, func() bool {
		snap := s.Snapshot()
		return snap.Schedules[1].Status == "fired"
	})
	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("runner took %v to wake after unfreeze (expected immediate)", elapsed)
	}
}

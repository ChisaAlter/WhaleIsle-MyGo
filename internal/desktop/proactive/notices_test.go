package proactive

import (
	"context"
	"sync/atomic"
	"testing"
)

func resultTask(t *testing.T, s *Service, dir string) Task {
	t.Helper()
	task := createTask(t, s, "汇总核对结果")
	task, err := s.PrepareExecution(task.ID, RunSpec{HostHome: "host", Cwd: dir}, false)
	if err != nil {
		t.Fatal(err)
	}
	task, err = s.updateExecution(task.ID, task.Execution.RequestID, "completed", "verifying", "已核对配置。仍需实际安装验收。", "")
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestResultNoticeAtomicDeliveryReplayAndUnknown(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	task := resultTask(t, s, dir)
	if len(s.Notices()) != 1 || s.Notices()[0].TaskID != task.ID || s.Notices()[0].Body != task.Execution.Summary {
		t.Fatal("completion not atomically delivered to inbox")
	}
	if _, err := s.updateExecution(task.ID, task.Execution.RequestID, "completed", "verifying", task.Execution.Summary, ""); err != nil {
		t.Fatal(err)
	}
	if len(s.Notices()) != 1 {
		t.Fatal("replayed completion duplicated notification")
	}
	var sent atomic.Int32
	deliver := func(context.Context, Notice) (string, error) { sent.Add(1); return "shown", nil }
	m := NewMonitor(context.Background(), s, nil, deliver, nil)
	waitFor(t, func() bool { return s.Notices()[0].DesktopStatus == "shown" })
	m.Close()
	s = openService(t, dir)
	m = NewMonitor(context.Background(), s, nil, deliver, nil)
	m.Close()
	if sent.Load() != 1 || s.Notices()[0].ReadAt != "" {
		t.Fatal("restart duplicated delivery or marked unread result read")
	}
	if err := s.ReadNotice(s.Notices()[0].ID); err != nil {
		t.Fatal(err)
	}
	if openService(t, dir).Notices()[0].ReadAt == "" {
		t.Fatal("read cursor did not persist")
	}
	second := resultTask(t, s, dir)
	id := "result:" + second.Execution.RequestID
	if _, err := s.ClaimNotice(id); err != nil {
		t.Fatal(err)
	}
	s = openService(t, dir)
	m = NewMonitor(context.Background(), s, nil, deliver, nil)
	waitFor(t, func() bool { return s.Notices()[1].DesktopStatus == "unknown" })
	m.Close()
	if sent.Load() != 1 {
		t.Fatal("unknown desktop submission silently retried")
	}
	if err := s.RetryNotice(id); err != nil {
		t.Fatal(err)
	}
	m = NewMonitor(context.Background(), s, nil, deliver, nil)
	waitFor(t, func() bool { return s.Notices()[1].DesktopStatus == "shown" })
	m.Close()
	if sent.Load() != 2 {
		t.Fatal("explicit delivery retry did not run")
	}
}

func TestUnavailableNoticeKeepsInAppResult(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	resultTask(t, s, dir)
	m := NewMonitor(context.Background(), s, nil, func(context.Context, Notice) (string, error) { return "unavailable", nil }, nil)
	waitFor(t, func() bool { return s.Notices()[0].DesktopStatus == "unavailable" })
	m.Close()
	s = openService(t, dir)
	if s.Notices()[0].Body == "" || s.Notices()[0].ReadAt != "" {
		t.Fatal("unsupported system notification lost unread result")
	}
}

func TestNoticeShutdownLeavesUnknownReceipt(t *testing.T) {
	dir := t.TempDir()
	s := openService(t, dir)
	resultTask(t, s, dir)
	started := make(chan struct{})
	m := NewMonitor(context.Background(), s, nil, func(ctx context.Context, n Notice) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}, nil)
	<-started
	m.Close()
	if s.Notices()[0].DesktopStatus != "sending" {
		t.Fatal("cancelled delivery claimed a result")
	}
	s = openService(t, dir)
	if err := s.recoverNotices(); err != nil {
		t.Fatal(err)
	}
	if s.Notices()[0].DesktopStatus != "unknown" || s.Notices()[0].ReadAt != "" {
		t.Fatal("shutdown lost unknown unread delivery")
	}
}

package harness

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"whaleisle/internal/desktop/taskcontrol"
)

func TestHostProtectionMemoryBoundary(t *testing.T) {
	const token = "fixture-only-task-token"
	t.Setenv("DSHD_TASK_CONTROL_TOKEN", "stale-fixture-token")
	env := childEnv(LaunchOptions{ExtraEnv: []string{"DSHD_TASK_CONTROL_TOKEN=" + token}})
	count := 0
	for _, item := range env {
		if strings.HasPrefix(item, "DSHD_TASK_CONTROL_TOKEN=") {
			count++
			if item != "DSHD_TASK_CONTROL_TOKEN="+token {
				t.Fatal("inherited stale task token")
			}
		}
	}
	if count != 1 {
		t.Fatalf("task token count = %d", count)
	}
	s := &Supervisor{taskControlToken: token, status: Status{State: Stopped}}
	s.scan(strings.NewReader("diagnostic "+token+"\n"), make(chan string, 1), token)
	if strings.Contains(strings.Join(s.logs, "\n"), token) {
		t.Fatal("task token leaked into logs")
	}
	data, err := json.Marshal(s.status)
	if err != nil || strings.Contains(string(data), token) {
		t.Fatal("task token leaked into status")
	}
	committed := false
	did, err := s.ProtectTaskAction(context.Background(), "quit", func() int { return 1 }, func(i taskcontrol.Inspection) (bool, error) {
		if i.LocalRunning != 1 {
			t.Error("local session missing")
		}
		return false, nil
	}, func() error { committed = true; return nil })
	if did || committed || err != taskcontrol.ErrCancelled {
		t.Fatalf("stopped Host bypassed local consent: %v %v", did, err)
	}
	s.nextGenLocked()
	if s.taskControlToken != "" {
		t.Fatal("stale token retained after generation change")
	}
}

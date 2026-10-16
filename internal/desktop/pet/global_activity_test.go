package pet

import "testing"

func TestPetGlobalActivity(t *testing.T) {
	idle := TaskStatus{State: "idle", SessionID: "selected", Title: "Selected"}
	got := taskWithGlobalActivity(idle, "host1", "working")
	if got.State != "running" || got.SessionID != "" || got.Title == idle.Title {
		t.Fatalf("global work: %+v", got)
	}
	for _, state := range []string{"idle", "unknown", "offline", ""} {
		if got := taskWithGlobalActivity(idle, "host1", state); got != idle {
			t.Fatalf("fallback %s: %+v", state, got)
		}
	}
	if got := taskWithGlobalActivity(idle, "", "working"); got != idle {
		t.Fatalf("cleared owner: %+v", got)
	}
	running := TaskStatus{State: "running", SessionID: "selected", Title: "Running"}
	if got := taskWithGlobalActivity(running, "host1", "working"); got != running {
		t.Fatalf("selected target lost: %+v", got)
	}
	p := &Companion{}
	p.task = idle
	p.notifications.owner = "host1"
	p.notifications.snapshot.State = "working"
	if got := p.activityTaskLocked(); got.State != "running" {
		t.Fatalf("watcher not connected: %+v", got)
	}
	p.notifications = notificationState{}
	if got := p.activityTaskLocked(); got != idle {
		t.Fatalf("owner clear retained activity: %+v", got)
	}
}

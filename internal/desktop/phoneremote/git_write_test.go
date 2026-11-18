package phoneremote

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestGitWriteRevocationKeepsRunningOwnerAndCancelsQueue(t *testing.T) {
	r := NewRuntime()
	child := newSyncHostChild()
	r.child = child
	running, cancelRunning := context.WithCancel(context.Background())
	queued, cancelQueued := context.WithCancel(context.Background())
	defer cancelRunning()
	defer cancelQueued()
	var released atomic.Int32
	r.gitWrites = map[string]*gitWriteAdmission{
		"1": {child: child, cancel: cancelRunning, release: func() { released.Add(1) }},
		"2": {child: child, cancel: cancelQueued},
	}
	r.RevokeGit()
	if queued.Err() == nil || running.Err() != nil || released.Load() != 0 {
		t.Fatal("revocation must cancel only waiting ownership")
	}
	r.releaseGitWrite(child, "1")
	r.releaseGitWrite(child, "1")
	if running.Err() == nil || released.Load() != 1 {
		t.Fatal("completed write did not release exactly once")
	}
	r.releaseChildGitWrites(child)
	if len(r.gitWrites) != 0 {
		t.Fatal("exited child retained pending ownership")
	}
}

func TestStopDoesNotKillAnAcceptedGitWrite(t *testing.T) {
	r := NewRuntime()
	reader, writer := io.Pipe()
	child := &fixtureChild{output: reader, writer: writer, done: make(chan struct{})}
	r.stopTimeout = time.Millisecond
	r.spawn = func(Resources) (Child, error) { return child, nil }
	r.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	r.Start("fixture", "fixture", map[string]any{"remoteEnabled": true})
	defer child.Kill()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	var released atomic.Int32
	r.gitWrites = map[string]*gitWriteAdmission{"1": {child: child, cancel: cancel, release: func() { released.Add(1) }}}
	stopped := r.Stop()
	if stopped == nil {
		t.Fatal("pending accepted write has no child-exit notification")
	}
	if state := r.Snapshot(); state.Status != "stopping" || state.Message == "" {
		t.Fatalf("accepted operation was hidden as closed: %+v", state)
	}
	if released.Load() != 0 {
		t.Fatal("stop released a still-running write")
	}
	select {
	case <-child.done:
		t.Fatal("stop killed the child while its accepted write was running")
	case <-stopped:
		t.Fatal("stop notification preceded child exit")
	default:
	}
	r.releaseGitWrite(child, "1")
	select {
	case <-stopped:
		t.Fatal("write release was mistaken for child exit")
	default:
	}
	child.Kill()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("actual child exit did not notify the pending application")
	}
	if released.Load() != 1 || r.Snapshot().Status != "offline" {
		t.Fatal("completed write did not allow stop to settle")
	}
	if pending := r.Stop(); pending != nil {
		t.Fatal("settled stop retained a pending notification")
	}
}

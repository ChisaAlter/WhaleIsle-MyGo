package botnotify

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func notifyFixture() Snapshot {
	return Snapshot{Revision: 1, Rows: []Row{{ID: "b", SessionID: "s", Name: "Bot", Notify: true, ActivitySeq: 1}}}
}
func TestBotNotifyEdgesAndSuppression(t *testing.T) {
	var tr Tracker
	s := notifyFixture()
	if n := tr.Observe(s, false, true); len(n) != 0 {
		t.Fatal("initial snapshot notified")
	}
	s.Rows[0].Attention = true
	s.Rows[0].GoalPhase = "blocked"
	s.Rows[0].Run = Run{RoutineID: "r", RunID: "r1", Status: "completed", EndedAt: 10}
	notices := tr.Observe(s, false, true)
	if len(notices) != 1 || len(notices[0].Events) != 3 {
		t.Fatalf("missing exact edges: %#v", notices)
	}
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("repeated status notified")
	}
	s.Rows[0].Run = Run{}
	tr.Observe(s, false, true)
	s.Rows[0].Run = Run{RoutineID: "r", RunID: "r1", Status: "completed", EndedAt: 10}
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("same terminal run notified twice")
	}
	for _, gate := range []string{"focused", "disabled", "unsupported"} {
		var muted Tracker
		base := notifyFixture()
		muted.Observe(base, false, true)
		base.Rows[0].Attention = true
		if gate == "disabled" {
			base.Rows[0].Notify = false
		}
		if len(muted.Observe(base, gate == "focused", gate != "unsupported")) != 0 {
			t.Fatal(gate)
		}
		base.Rows[0].Notify = true
		if len(muted.Observe(base, false, true)) != 0 {
			t.Fatal("deferred replay after " + gate)
		}
	}
}
func TestBotNotifyIdentityRevisionAndRuns(t *testing.T) {
	var tr Tracker
	s := notifyFixture()
	tr.Observe(s, false, true)
	s.Rows[0].Attention = true
	n := tr.Observe(s, false, true)[0]
	if !tr.Current(n.Ticket) {
		t.Fatal("new ticket rejected")
	}
	s.Revision = 2
	tr.Observe(s, false, true)
	if tr.Current(n.Ticket) {
		t.Fatal("old catalog ticket survived")
	}
	s.Rows[0].SessionID = "new-session"
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("canonical replacement notified")
	}
	s.Revision = 1
	s.Rows[0].Attention = false
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("stale catalog accepted")
	}
	s.Revision = 2
	s.Rows[0].Attention = true
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("stale result changed baseline")
	}
	s.Rows[0].Run = Run{RoutineID: "r", RunID: "silent", Status: "completed", Silent: true, EndedAt: 20}
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("silent run notified")
	}
	s.Rows[0].Run = Run{RoutineID: "r", RunID: "running", Status: "running", EndedAt: 30}
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("running notified")
	}
	for i, status := range []string{"completed", "failed", "interrupted"} {
		s.Rows[0].Run = Run{RoutineID: "r", RunID: status, Status: status, EndedAt: int64(40 + i)}
		if len(tr.Observe(s, false, true)) != 1 {
			t.Fatal(status)
		}
	}
	tr.Clear()
	if tr.Current(n.Ticket) || len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("clear did not fence baseline")
	}
}
func TestBotNotifyBoundedPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var active, max, calls atomic.Int32
	release := make(chan struct{})
	done := make(chan struct{})
	read := func(ctx context.Context) (Snapshot, error) {
		n := active.Add(1)
		if n > max.Load() {
			max.Store(n)
		}
		calls.Add(1)
		defer active.Add(-1)
		select {
		case <-release:
			return Snapshot{}, errors.New("fake failure")
		case <-ctx.Done():
			return Snapshot{}, ctx.Err()
		}
	}
	results := make(chan error, 1)
	go func() {
		defer close(done)
		Poll(ctx, time.Millisecond, time.Second, read, func(_ Snapshot, err error) { results <- err })
	}()
	for calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(5 * time.Millisecond)
	if calls.Load() != 1 || max.Load() != 1 {
		t.Fatal("read polling overlapped")
	}
	close(release)
	select {
	case err := <-results:
		if err == nil {
			t.Fatal("error hidden")
		}
	case <-time.After(time.Second):
		t.Fatal("no failure result")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("poll did not stop")
	}
	if !reflect.DeepEqual([]int{MaxBots, int(Interval / time.Millisecond)}, []int{200, 1500}) {
		t.Fatal("original bounds changed")
	}
}

func TestBotNotifyExactClickTargetsAndStaleActivity(t *testing.T) {
	var tr Tracker
	s := notifyFixture()
	s.Rows = append(s.Rows, Row{ID: "hidden-room", SessionID: "room-session", Kind: "room", Hidden: true, Notify: true, ActivitySeq: 8})
	tr.Observe(s, false, true)
	s.Rows[1].Attention = true
	notices := tr.Observe(s, false, true)
	if len(notices) != 1 {
		t.Fatal("hidden/room event was filtered")
	}
	ticket := notices[0].Ticket
	target, ok := tr.Target(ticket)
	if !ok || target.ID != "hidden-room" || target.SessionID != "room-session" || !target.Hidden || target.Kind != "room" {
		t.Fatal("click target lost exact identity")
	}
	s.Rows[1].ActivitySeq = 7
	s.Rows[1].Attention = false
	tr.Observe(s, false, true)
	if !tr.Current(ticket) {
		t.Fatal("stale activity overwrote state")
	}
	s.Rows[1].ActivitySeq = 8
	s.Rows[1].Attention = true
	if len(tr.Observe(s, false, true)) != 0 {
		t.Fatal("stale response manufactured second attention")
	}
	s.Rows = s.Rows[:1]
	tr.Observe(s, false, true)
	if _, ok := tr.Target(ticket); ok {
		t.Fatal("deleted target remained clickable")
	}
}

package startup

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestReadinessTransitions(t *testing.T) {
	tr := NewTracker()
	var mu sync.Mutex
	var seen []Level
	tr.OnChange(func(r Readiness) {
		mu.Lock()
		seen = append(seen, r.Level)
		mu.Unlock()
	})
	if tr.Current().Level != NotReady {
		t.Fatal("a fresh tracker is NotReady")
	}
	if tr.Reach(1, EventsReady) {
		t.Fatal("events cannot be ready before a transport")
	}
	tr.Begin(1)
	if tr.Reach(1, WorkbenchReady) {
		t.Fatal("the workbench cannot be ready before $events")
	}
	if !tr.Reach(1, EventsReady) || !tr.Reach(1, WorkbenchReady) {
		t.Fatal("levels rise in order")
	}
	if tr.Reach(1, EventsReady) {
		t.Fatal("a lower level never lowers the published one")
	}
	first := tr.Current().Reached[WorkbenchReady]
	tr.Lose(1)
	if got := tr.Current().Level; got != TransportReady {
		t.Fatalf("carrier loss leaves %s, want transport_ready", got)
	}
	tr.Reach(1, EventsReady)
	tr.Reach(1, WorkbenchReady)
	if tr.Current().Reached[WorkbenchReady] != first {
		t.Fatal("the first WorkbenchReady time survives a reconnect")
	}
	tr.Begin(2)
	if tr.Reach(1, EventsReady) {
		t.Fatal("a superseded generation cannot raise the new one")
	}
	if r := tr.Current(); r.Generation != 2 || r.Level != TransportReady {
		t.Fatalf("new generation at %+v", r)
	}
	tr.Reset()
	if tr.Current().Level != NotReady {
		t.Fatal("Reset drops to NotReady")
	}
	mu.Lock()
	defer mu.Unlock()
	want := []Level{TransportReady, EventsReady, WorkbenchReady, TransportReady, EventsReady, WorkbenchReady, TransportReady, NotReady}
	if len(seen) != len(want) {
		t.Fatalf("transitions %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("transitions %v, want %v", seen, want)
		}
	}
}

func newTestScheduler() (*Tracker, *Scheduler) {
	tr := NewTracker()
	s := NewScheduler(tr)
	s.Stagger, s.Idle, s.Fallback = 5*time.Millisecond, 80*time.Millisecond, 0
	return tr, s
}

func TestTierBWaitsForWorkbenchAndStaggers(t *testing.T) {
	tr, s := newTestScheduler()
	s.Stagger = 20 * time.Millisecond
	tr.Begin(1)
	var mu sync.Mutex
	var at []time.Time
	for _, name := range []string{"model.catalog", "agent.presets", "skills"} {
		s.AfterWorkbench(context.Background(), 1, name, func() {
			mu.Lock()
			at = append(at, time.Now())
			mu.Unlock()
		})
	}
	tr.Reach(1, EventsReady)
	time.Sleep(30 * time.Millisecond)
	if s.Started("model.catalog") != 0 {
		t.Fatal("Tier B must not start before WorkbenchReady")
	}
	tr.Reach(1, WorkbenchReady)
	eventually(t, "three Tier B jobs", func() bool { mu.Lock(); defer mu.Unlock(); return len(at) == 3 })
	mu.Lock()
	defer mu.Unlock()
	for i := 1; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < 15*time.Millisecond {
			t.Fatalf("Tier B jobs %d and %d only %s apart", i-1, i, gap)
		}
	}
}

func TestTierBDroppedWithItsGeneration(t *testing.T) {
	tr, s := newTestScheduler()
	tr.Begin(1)
	var ran atomic.Int32
	s.AfterWorkbench(context.Background(), 1, "vision", func() { ran.Add(1) })
	tr.Reset()
	tr.Begin(2)
	tr.Reach(2, EventsReady)
	tr.Reach(2, WorkbenchReady)
	time.Sleep(30 * time.Millisecond)
	if ran.Load() != 0 {
		t.Fatal("a stopped generation's Tier B job must not run for the next Host")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.AfterWorkbench(ctx, 2, "vision", func() { ran.Add(1) })
	time.Sleep(30 * time.Millisecond)
	if ran.Load() != 0 {
		t.Fatal("a cancelled Tier B job must not run")
	}
}

func TestTierBFallbackWhenWorkbenchNeverReady(t *testing.T) {
	tr, s := newTestScheduler()
	s.Fallback = 20 * time.Millisecond
	tr.Begin(1)
	var ran atomic.Int32
	s.AfterWorkbench(context.Background(), 1, "usage.billing", func() { ran.Add(1) })
	tr.Reach(1, EventsReady)
	eventually(t, "fallback release", func() bool { return ran.Load() == 1 })
}

func TestTierCNotStartedBeforeTrigger(t *testing.T) {
	tr, s := newTestScheduler()
	s.Idle = time.Hour
	var cold, update atomic.Int32
	s.Deferred("im.store", func() { cold.Add(1) })
	s.Deferred("update.center", func() { update.Add(1) })
	tr.Begin(1)
	tr.Reach(1, EventsReady)
	tr.Reach(1, WorkbenchReady)
	time.Sleep(30 * time.Millisecond)
	if cold.Load() != 0 || update.Load() != 0 || s.Started("im.store") != 0 {
		t.Fatal("Tier C must not start before its first use or the idle window")
	}
	s.Use("assistant.store") // same group as im.store
	eventually(t, "cold stores on first use", func() bool { return cold.Load() == 1 })
	s.Use("im.store")
	time.Sleep(10 * time.Millisecond)
	if cold.Load() != 1 || s.Started("assistant.store") != 1 {
		t.Fatal("a Tier C group runs once per process")
	}
	if update.Load() != 0 {
		t.Fatal("first use of one group must not start another")
	}
}

func TestTierCStartsAfterIdle(t *testing.T) {
	tr, s := newTestScheduler()
	var ran atomic.Int32
	s.Deferred("whale.bridge.alerts", func() { ran.Add(1) })
	time.Sleep(120 * time.Millisecond)
	if ran.Load() != 0 {
		t.Fatal("the idle window starts at WorkbenchReady, not process start")
	}
	tr.Begin(1)
	tr.Reach(1, EventsReady)
	tr.Reach(1, WorkbenchReady)
	eventually(t, "idle start", func() bool { return ran.Load() == 1 })
	var late atomic.Int32
	s.Deferred("phone.gateway", func() { late.Add(1) })
	eventually(t, "late registration after idle", func() bool { return late.Load() == 1 })
}

func TestTierCUseBeforeRegistration(t *testing.T) {
	_, s := newTestScheduler()
	s.Use("update.center")
	var ran atomic.Int32
	s.Deferred("update.center", func() { ran.Add(1) })
	eventually(t, "use-before-register", func() bool { return ran.Load() == 1 })
}

func TestPlanRowsAreComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range Plan {
		if row.Name == "" || row.Owner == "" || row.Trigger == "" {
			t.Fatalf("incomplete row %+v", row)
		}
		if seen[row.Name] {
			t.Fatalf("duplicate row %s", row.Name)
		}
		seen[row.Name] = true
		if row.Tier == TierOS && row.Why == "" {
			t.Fatalf("%s starts with the process and must say why", row.Name)
		}
	}
	_, s := newTestScheduler()
	var ran atomic.Int32
	s.Deferred("not.in.plan", func() { ran.Add(1) })
	s.AfterWorkbench(context.Background(), 1, "im.store", func() { ran.Add(1) }) // wrong tier
	s.Use("not.in.plan")
	time.Sleep(10 * time.Millisecond)
	if ran.Load() != 0 {
		t.Fatal("jobs outside the plan, or in the wrong tier, never run")
	}
}

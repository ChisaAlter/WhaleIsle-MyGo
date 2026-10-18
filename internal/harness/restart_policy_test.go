package harness

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSP06RestartPolicySnapshotOutsideSupervisorLock(t *testing.T) {
	var s *Supervisor
	latest := RestartPolicy{AutoRestart: true, MaxAttempts: 5, BaseDelay: 2 * time.Second}
	reads := 0
	s = New(Options{AutoRestart: false, MaxAttempts: 1, BaseDelay: time.Second, ReadRestartPolicy: func() RestartPolicy {
		// A provider can inspect status without waiting on a lock held by its caller.
		_ = s.Status()
		reads++
		return latest
	}})
	s.status = Status{State: Ready, Generation: 7}
	s.gen = 7
	s.attempt = 2
	done := make(chan RestartPolicy, 1)
	go func() { done <- s.RestartPolicy() }()
	select {
	case p := <-done:
		if p != latest {
			t.Fatalf("snapshot %+v", p)
		}
	case <-time.After(time.Second):
		t.Fatal("policy callback called under supervisor lock")
	}
	if reads != 1 || s.Status().State != Ready || s.gen != 7 || s.attempt != 2 {
		t.Fatal("policy read changed lifecycle")
	}
	latest = RestartPolicy{false, 1, 500 * time.Millisecond}
	s.fail(7, errors.New("synthetic failure"), "fake")
	if s.Status().State != Failed || s.Status().Attempt != 2 || reads != 2 {
		t.Fatalf("new failure ignored committed disable: %+v; reads %d", s.Status(), reads)
	}
}

func TestSP06RestartDecisionUsesOneLatestSnapshot(t *testing.T) {
	latest := RestartPolicy{true, 10, 30 * time.Second}
	reads := 0
	s := New(Options{AutoRestart: false, MaxAttempts: 1, BaseDelay: time.Millisecond, ReadRestartPolicy: func() RestartPolicy { reads++; return latest }})
	s.gen = 4
	s.attempt = 9
	s.fail(4, errors.New("synthetic failure"), "fake")
	// The selected delay is over four hours. Invalidate that fake generation
	// immediately, so the timer can never launch a child even in a long test run.
	s.mu.Lock()
	generation := s.gen
	s.nextGenLocked()
	s.mu.Unlock()
	if st := s.Status(); st.State != Starting || st.Attempt != 10 || generation != 5 || reads != 1 {
		t.Fatalf("decision %+v, reads %d", st, reads)
	}
	if logs := strings.Join(s.Logs(), "\n"); !strings.Contains(logs, "restart 10/10 in 4h16m0s") {
		t.Fatalf("did not consume one coherent snapshot: %s", logs)
	}
	latest = RestartPolicy{true, 2, time.Second}
	s.fail(s.gen, errors.New("synthetic second failure"), "fake")
	if st := s.Status(); st.State != Failed || st.Attempt != 10 || reads != 2 {
		t.Fatalf("new maximum ignored: %+v reads %d", st, reads)
	}
}

func TestSP06RestartPolicyBackoffAndFallback(t *testing.T) {
	p := RestartPolicy{true, 3, 500 * time.Millisecond}
	for attempt, want := range []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second} {
		if got, ok := p.retryDelay(attempt); !ok || got != want {
			t.Fatalf("attempt %d: %s %v", attempt, got, ok)
		}
	}
	if _, ok := p.retryDelay(3); ok {
		t.Fatal("attempt limit ignored")
	}
	p.AutoRestart = false
	if _, ok := p.retryDelay(0); ok {
		t.Fatal("disabled policy retried")
	}
	s := New(Options{AutoRestart: true, MaxAttempts: 2})
	if got := s.RestartPolicy(); got != (RestartPolicy{true, 2, time.Second}) {
		t.Fatalf("legacy options fallback: %+v", got)
	}
}

func TestSP06PolicyReadDoesNotInvertConfigLockOrder(t *testing.T) {
	var configMu sync.Mutex
	entered := make(chan struct{})
	s := New(Options{ReadRestartPolicy: func() RestartPolicy { close(entered); configMu.Lock(); defer configMu.Unlock(); return RestartPolicy{} }})
	configMu.Lock()
	done := make(chan struct{})
	go func() { _ = s.RestartPolicy(); close(done) }()
	<-entered
	statusDone := make(chan struct{})
	go func() { _ = s.Status(); close(statusDone) }()
	select {
	case <-statusDone:
	case <-time.After(time.Second):
		configMu.Unlock()
		t.Fatal("provider waited on config while holding supervisor mutex")
	}
	configMu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("policy read failed to finish")
	}
}

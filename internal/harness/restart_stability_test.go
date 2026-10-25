package harness

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReadyThenCrashExhaustsRestartBudget(t *testing.T) {
	s := supervisorFor(t, "flap", Options{AutoRestart: true, MaxAttempts: 2, BaseDelay: 10 * time.Millisecond})
	t.Cleanup(s.Stop)
	s.Start()
	st := waitFor(t, s, Failed)
	if st.Attempt != 2 || st.Err == nil {
		t.Fatalf("short-lived Ready must exhaust its budget: %+v", st)
	}
	if n := strings.Count(strings.Join(s.Logs(), "\n"), "restart "); n != 2 {
		t.Fatalf("got %d restarts, want 2; logs:\n%s", n, strings.Join(s.Logs(), "\n"))
	}
}

func TestRestartBudgetRecoversOnlyAfterStableReady(t *testing.T) {
	for _, tc := range []struct {
		name    string
		age     time.Duration
		want    State
		attempt int
	}{
		{"brief-ready", time.Second, Failed, 2},
		{"stable-ready", restartStablePeriod + time.Second, Starting, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Long timers are cancelled without ever launching a child. Advancing
			// the owned Ready timestamp avoids a minute-long fixture wait.
			s := New(Options{AutoRestart: true, MaxAttempts: 2, BaseDelay: time.Hour})
			t.Cleanup(s.Stop)
			s.gen, s.attempt = 4, 2
			s.setLocked(Status{State: Ready})
			s.readyAt = time.Now().Add(-tc.age)
			s.fail(4, errors.New("synthetic online crash"), "fixture")
			if st := s.Status(); st.State != tc.want || st.Attempt != tc.attempt {
				t.Fatalf("got %+v, want %s attempt %d", st, tc.want, tc.attempt)
			}
		})
	}
}

func TestStaleGenerationCannotResetRestartBudget(t *testing.T) {
	s := New(Options{AutoRestart: true, MaxAttempts: 2, BaseDelay: time.Hour})
	s.gen, s.attempt = 4, 2
	s.setLocked(Status{State: Ready})
	s.readyAt = time.Now().Add(-2 * restartStablePeriod)
	before := s.Status()
	s.fail(3, errors.New("old process crash"), "fixture")
	if s.Status() != before || s.attempt != 2 {
		t.Fatal("stale process changed the current restart budget")
	}
	s.mu.Lock()
	s.nextGenLocked()
	s.mu.Unlock()
	if !s.readyAt.IsZero() {
		t.Fatal("replacement generation inherited a stable Ready period")
	}
}

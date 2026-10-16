package harness

import (
	"errors"
	"os/exec"
	"testing"
	"time"
)

// Fixtures use long timers and inert process handles; they never start a Host.
func TestCancelRecoveryInvalidatesQueuedRetry(t *testing.T) {
	s := New(Options{AutoRestart: true, MaxAttempts: 3, BaseDelay: time.Hour})
	s.gen = 1
	failure := errors.New("synthetic crash")
	s.fail(1, failure, "fixture")
	queued := s.Status()
	if !queued.RecoveryActive || !s.recoveryQueued {
		t.Fatal("retry not cancellable")
	}
	st := s.CancelRecovery()
	if st.State != Failed || !st.RecoveryCancelled || st.RecoveryActive || st.Err != failure || st.Source != "fixture" {
		t.Fatalf("cancelled snapshot: %+v", st)
	}
	if s.current(queued.Generation) || s.restartTimer != nil {
		t.Fatal("queued retry still valid")
	}
	// Even an already-dispatched timer cannot revive the cancelled generation.
	s.run(queued.Generation)
	if s.Status() != st || s.cmd != nil {
		t.Fatal("stale timer revived recovery")
	}
	if got := s.CancelRecovery(); got != st {
		t.Fatal("cancel is not idempotent")
	}
	if !s.opts.AutoRestart {
		t.Fatal("cancel changed persistent policy")
	}
}

func TestCancelRecoveryKeepsClaimedChildOwned(t *testing.T) {
	s := New(Options{AutoRestart: true, MaxAttempts: 3, BaseDelay: time.Hour})
	s.gen = 7
	s.recoveryActive = true
	s.cmd = &exec.Cmd{} // inert handle: cancellation must not call terminate/Stop
	owned := s.cmd
	s.exited = make(chan struct{})
	s.taskControlToken = "fixture-token"
	s.setLocked(Status{State: Starting})
	st := s.CancelRecovery()
	if st.State != Starting || !st.RecoveryCancelled || s.gen != 7 || s.cmd != owned || s.exited == nil || s.taskControlToken != "fixture-token" {
		t.Fatal("cancellation disturbed the claimed child")
	}
	// Cancellation suppresses both normal and plugin-recovery retries.
	s.startLogs = []string{"plugin tree failed to load"}
	s.fail(7, errors.New("synthetic crash"), "fixture")
	if st := s.Status(); st.State != Failed || !st.RecoveryCancelled || st.RecoveryActive || s.gen != 7 || s.opts.Launch.SkipUserPlugins {
		t.Fatalf("cancelled recovery scheduled more work: %+v", st)
	}
}

func TestCancelRecoveryDoesNotCancelManualStartOrReady(t *testing.T) {
	for _, state := range []State{Stopped, Starting, Ready, Failed} {
		s := New(Options{})
		s.gen = 9
		s.setLocked(Status{State: state})
		before := s.Status()
		if got := s.CancelRecovery(); got != before || s.gen != 9 {
			t.Fatalf("changed manual state %v", state)
		}
	}
}

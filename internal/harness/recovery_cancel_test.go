package harness

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Fixtures use long timers and a sleeping test process; they never start a Host.
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

func TestCancelRecoveryTerminatesClaimedChild(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "FAKE_DSH=hold")
	detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	t.Cleanup(func() { _ = terminate(cmd, true); <-exited })
	s := New(Options{AutoRestart: true, MaxAttempts: 3, BaseDelay: time.Hour})
	s.gen = 7
	s.recoveryActive = true
	s.cmd, s.exited = cmd, exited
	s.taskControlToken = "fixture-token"
	s.setLocked(Status{State: Starting})
	st := s.CancelRecovery()
	if st.State != Failed || !st.RecoveryCancelled || s.gen == 7 || s.cmd != nil || s.exited != nil || s.taskControlToken != "" {
		t.Fatalf("claimed recovery remains active: %+v", st)
	}
	select {
	case <-exited:
	default:
		t.Fatal("cancel returned before child exit")
	}
	s.fail(7, errors.New("plugin tree failed to load"), "fixture")
	s.run(7)
	if s.Status() != st || s.restartTimer != nil || !s.opts.AutoRestart || s.opts.Launch.SkipUserPlugins {
		t.Fatal("cancelled attempt revived or changed persistent policy")
	}
}

func TestCancelRecoveryInvalidatesClaimedPreparation(t *testing.T) {
	s := New(Options{})
	s.gen, s.recoveryActive = 7, true
	s.setLocked(Status{State: Starting})
	st := s.CancelRecovery()
	s.run(7)
	if s.current(7) || s.cmd != nil || s.Status() != st || st.State != Failed {
		t.Fatal("claimed preparation survived cancellation")
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

package harness

import (
	"errors"
	"strings"
	"testing"
)

func TestStartupRecoveryPersistenceFailureStopsRetry(t *testing.T) {
	called := 0
	s := New(Options{OnPluginRecovery: func(reason string, logs []string) error {
		called++
		if reason != "synthetic exit" || len(logs) != 1 {
			t.Fatalf("wrong evidence: %q %v", reason, logs)
		}
		return errors.New("synthetic read-only storage")
	}})
	s.gen = 1
	s.startLogs = []string{"plugin tree failed to load"}
	s.fail(1, errors.New("synthetic exit"), "fixture")
	st := s.Status()
	if called != 1 || st.State != Failed || !strings.Contains(st.Err.Error(), "could not be saved") {
		t.Fatalf("unexpected result: %+v, %d", st, called)
	}
	if s.opts.Launch.SkipUserPlugins || s.skippedOnce || s.gen != 1 {
		t.Fatal("started recovery despite persistence failure")
	}
}
func TestStartupRecoveryDoesNotBlamePreviousAttempt(t *testing.T) {
	s := New(Options{})
	s.gen = 1
	s.logs = []string{"plugin tree failed to load"}
	s.startLogs = []string{"unrelated current failure"}
	s.fail(1, errors.New("ordinary failure"), "fixture")
	if s.Status().State != Failed || s.opts.Launch.SkipUserPlugins {
		t.Fatal("historical log caused plugin skip")
	}
}
func TestStartupRecoveryStatusAndManualReset(t *testing.T) {
	s := New(Options{})
	s.skippedOnce = true
	s.SetSkipUserPlugins(true)
	s.mu.Lock()
	s.attemptRecovery = true
	s.setLocked(Status{State: Ready})
	s.mu.Unlock()
	if !s.Status().Recovery || !s.opts.Prepare.SkipUserPlugins {
		t.Fatal("recovery Ready omitted skip state")
	}
	s.SetSkipUserPlugins(false)
	if s.skippedOnce || s.opts.Prepare.SkipUserPlugins {
		t.Fatal("manual full retry did not reset skip guard")
	}
}

func TestStartupRecoveryIgnoresLateGenerationLogs(t *testing.T) {
	s := New(Options{})
	s.gen = 2
	s.status = Status{State: Starting}
	s.scan(strings.NewReader("plugin tree failed to load\n"), make(chan string, 1), "", 1)
	if len(s.CurrentStartLogs()) != 0 {
		t.Fatal("late prior generation polluted current evidence")
	}
}
func TestStartupRecoveryLiveFlagUnaffectedByNextLaunchSetting(t *testing.T) {
	s := New(Options{})
	s.attemptRecovery = true
	s.SetSkipUserPlugins(false)
	s.mu.Lock()
	s.setLocked(Status{State: Ready})
	s.mu.Unlock()
	if !s.Status().Recovery {
		t.Fatal("next-launch preference relabeled a recovered Host as full healthy")
	}
}

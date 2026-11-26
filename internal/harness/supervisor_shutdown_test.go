package harness

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Shutdown is bounded by its own budget, not by the Host's children. These
// tests drive the supervisor with the fake launcher: no window, no desktop.
const shutdownBound = stopGrace + stopExitBudget + 3*time.Second

func stopWithin(t *testing.T, s *Supervisor) time.Duration {
	t.Helper()
	returned := make(chan time.Duration, 1)
	go func() {
		started := time.Now()
		s.Stop()
		returned <- time.Since(started)
	}()
	select {
	case took := <-returned:
		return took
	case <-time.After(shutdownBound):
		t.Fatalf("Stop did not return within %s", shutdownBound)
		return 0
	}
}

func TestStopWhileStartingIsBounded(t *testing.T) {
	s := supervisorFor(t, "hold", Options{})
	s.Start()
	waitFor(t, s, Starting)
	took := stopWithin(t, s)
	if took > shutdownBound {
		t.Fatalf("Stop took %s", took)
	}
	if st := s.Status(); st.State != Stopped {
		t.Fatalf("after Stop: %v", st.State)
	}
}

func TestStopIsBoundedWhenAStatusListenerNeverReturns(t *testing.T) {
	blocked := make(chan struct{})
	t.Cleanup(func() { close(blocked) })
	s := supervisorFor(t, "serve", Options{OnChange: func(Status) { <-blocked }})
	s.Start()
	waitFor(t, s, Ready)

	took := stopWithin(t, s)
	if st := s.Status(); st.State != Stopped {
		t.Fatalf("after Stop: %v", st.State)
	}
	if took > shutdownBound {
		t.Fatalf("Stop took %s", took)
	}
}

func TestWaitExitReportsBudgetExpiry(t *testing.T) {
	never := make(chan struct{})
	if waitExit(never, 20*time.Millisecond) {
		t.Fatal("waitExit reported an exit that never happened")
	}
	closed := make(chan struct{})
	close(closed)
	if !waitExit(closed, time.Second) {
		t.Fatal("waitExit missed a closed channel")
	}
	if !waitExit(nil, time.Second) {
		t.Fatal("a supervisor with no child has nothing to wait for")
	}
}

func TestStopRemovesTheHostRecordOnlyAfterAConfirmedExit(t *testing.T) {
	record := filepath.Join(t.TempDir(), "host-process.json")
	s := supervisorFor(t, "serve", Options{RecordPath: record})
	s.Start()
	waitFor(t, s, Ready)
	if _, err := os.Stat(record); err != nil {
		t.Fatalf("Host was not recorded: %v", err)
	}
	took := stopWithin(t, s)
	if took > shutdownBound {
		t.Fatalf("Stop took %s", took)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("confirmed stop left the Host record behind: %v", err)
	}
}

func TestFailPreservesHostRecordWhenExitUnconfirmed(t *testing.T) {
	record := filepath.Join(t.TempDir(), "host-process.json")
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
	go func() { _ = cmd.Wait(); close(exited) }()
	t.Cleanup(func() { _ = terminate(cmd, true); <-exited })

	s := New(Options{RecordPath: record})
	s.mu.Lock()
	s.gen = 1
	s.cmd = cmd
	s.exited = exited
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	s.recordHost(cmd.Process.Pid, 47911, exe, 1)
	if _, err := os.Stat(record); err != nil {
		t.Fatalf("record not written: %v", err)
	}

	s.failWithExit(1, errors.New("timed out waiting for exit"), "fixture", false)
	if _, err := os.Stat(record); err != nil {
		t.Fatalf("unconfirmed exit removed host-process.json: %v", err)
	}
}

func TestCancelRecoveryCleansHostRecordAndLogSink(t *testing.T) {
	dir := t.TempDir()
	record := filepath.Join(dir, "host-process.json")
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
	job := bindChildJob(cmd)
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	t.Cleanup(func() { _ = terminateWithJob(cmd, job, true); job.close(); <-exited })

	sink := newHostLogSink(filepath.Join(dir, "logs"), cmd.Process.Pid)
	if sink == nil {
		t.Fatal("failed to open hostLogSink")
	}
	sink.add("before cancel")

	s := New(Options{RecordPath: record, AutoRestart: true, MaxAttempts: 2, BaseDelay: time.Hour})
	s.mu.Lock()
	s.gen = 5
	s.recoveryActive = true
	s.cmd, s.job, s.exited = cmd, job, exited
	s.logSink = sink
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	s.recordHost(cmd.Process.Pid, 47912, exe, 5)

	st := s.CancelRecovery()
	if st.State != Failed || !st.RecoveryCancelled {
		t.Fatalf("CancelRecovery status = %+v", st)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("CancelRecovery left host-process.json behind: %v", err)
	}
	select {
	case <-sink.done:
	case <-time.After(time.Second):
		t.Fatal("CancelRecovery did not close logSink")
	}
	if err := os.Remove(sink.path); err != nil {
		t.Fatalf("logSink file handle still open after CancelRecovery: %v", err)
	}
}

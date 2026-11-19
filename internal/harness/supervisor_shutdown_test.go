package harness

import (
	"os"
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

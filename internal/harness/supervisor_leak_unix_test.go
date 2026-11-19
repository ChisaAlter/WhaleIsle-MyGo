//go:build !windows

package harness

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// A Host that spawns a daemon inheriting fd 1 leaves the shell holding a pipe no
// one will close. Closing the window must not wait for that pipe: the shell has
// to reap the Host and finish, leaving the escaped daemon to the next launch's
// orphan reap.
func TestStopFinishesWhenAHostDescendantHoldsStdout(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "leaker.pid")
	t.Setenv("FAKE_DSH_LEAKER_PID", pidFile)
	t.Cleanup(func() { killLeaker(t, pidFile) })

	s := supervisorFor(t, "leak", Options{})
	s.Start()
	waitFor(t, s, Ready)

	s.mu.Lock()
	hostPID := s.cmd.Process.Pid
	s.mu.Unlock()

	took := stopWithin(t, s)
	if took > shutdownBound {
		t.Fatalf("Stop took %s with a descendant holding stdout", took)
	}
	if st := s.Status(); st.State != Stopped {
		t.Fatalf("after Stop: %v", st.State)
	}
	if err := syscall.Kill(hostPID, 0); err != syscall.ESRCH {
		t.Fatalf("Host %d still exists after Stop: %v", hostPID, err)
	}
	leaker := readLeakerPID(t, pidFile)
	if err := syscall.Kill(leaker, 0); err != nil {
		t.Fatalf("the fixture's escaped descendant is gone, so the pipe was never held: %v", err)
	}
}

func readLeakerPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(path); err == nil {
			if pid, convErr := strconv.Atoi(string(raw)); convErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the fake Host never recorded its escaped descendant in %s", path)
	return 0
}

func killLeaker(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 0 {
		return
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		t.Logf("kill leaked descendant %d: %v", pid, err)
	}
}

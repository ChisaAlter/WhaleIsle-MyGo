//go:build windows

package harness

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// startStdoutLeaker has no Windows implementation: a child started here stays in
// the Host's job tree, so the leaked-pipe fixture is a Unix test. The bounded
// waits it guards are covered on every platform by
// TestStopIsBoundedWhenAStatusListenerNeverReturns and TestWaitExitReportsBudgetExpiry.
func startStdoutLeaker() {}

func TestWindowsJobKillsGrandchildAfterParentExitsEarly(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	t.Setenv("FAKE_DSH_GRANDCHILD_PID", pidFile)

	s := supervisorFor(t, "earlyparent", Options{})
	defer s.Stop()
	s.Start()
	waitFor(t, s, Failed)

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("grandchild pid file missing: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid grandchild pid %q: %v", raw, err)
	}

	// Because the Host child was bound to a KILL_ON_JOB_CLOSE Job Object and
	// failWithExit terminates+closes the job when the parent exits, the
	// surviving grandchild must already be terminated even though its parent
	// exited first (where taskkill /PID <dead> /T /F would fail).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		h, openErr := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		if openErr != nil {
			return // process is gone
		}
		status, waitErr := windows.WaitForSingleObject(h, 0)
		_ = windows.CloseHandle(h)
		if waitErr == nil && status == windows.WAIT_OBJECT_0 {
			return // process exited
		}
		time.Sleep(25 * time.Millisecond)
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
	t.Fatalf("grandchild PID %d survived after parent exited early and supervisor handled failure", pid)
}

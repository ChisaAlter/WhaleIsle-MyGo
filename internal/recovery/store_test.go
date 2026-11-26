package recovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRestartAndConflictingInstance(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Update(func(s *State) {
		s.Workspace = "/project"
		s.SelectedSession = "session-1"
		s.Layout = Layout{Sidebar: 300, Right: 450, RightOpen: true}
		s.Drafts = map[string]json.RawMessage{"session-1": json.RawMessage(`{"text":"unfinished draft"}`)}
		s.Files = json.RawMessage(`{"dirty":"unsaved buffer"}`)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Update(func(s *State) { s.Workspace = "stale" }); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale writer: %v", err)
	}
	resumed, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := resumed.Snapshot()
	if got.Workspace != "/project" || got.SelectedSession != "session-1" || got.Layout.Sidebar != 300 || !bytes.Contains(got.Drafts["session-1"], []byte("unfinished")) || !bytes.Contains(got.Files, []byte("unsaved")) {
		t.Fatalf("resume lost state: %+v", got)
	}
	got.Drafts["session-1"][0] = 'x'
	if !json.Valid(resumed.Snapshot().Drafts["session-1"]) {
		t.Fatal("snapshot aliased")
	}
}

func TestInvalidAndLockedFilesArePreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	broken := []byte(`{"drafts": incomplete`)
	if err := os.WriteFile(path, broken, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("corrupt file accepted")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, broken) {
		t.Fatal("corrupt original changed")
	}
	os.Remove(path)
	s, _ := Open(dir)
	if err := os.WriteFile(path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(v *State) { v.Workspace = "blocked" }); !errors.Is(err, ErrBusy) {
		t.Fatalf("locked writer: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("locked write changed disk")
	}
	if s.Snapshot().Workspace != "" {
		t.Fatal("failed write changed memory")
	}
}

func TestLockFileContainsOwnerDiagnosticsAndRespectsErrBusy(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, FileName+".lock")
	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(lockPath)
	if err != nil {
		release()
		t.Fatal(err)
	}
	var info lockOwnerInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		release()
		t.Fatalf("invalid lock diagnostic payload %q: %v", raw, err)
	}
	if info.PID != os.Getpid() {
		release()
		t.Fatalf("lock pid = %d, want %d", info.PID, os.Getpid())
	}
	if _, err := time.Parse(time.RFC3339Nano, info.CreatedAt); err != nil {
		release()
		t.Fatalf("invalid lock timestamp %q: %v", info.CreatedAt, err)
	}

	if _, err := acquireLock(lockPath); !errors.Is(err, ErrBusy) {
		release()
		t.Fatalf("second acquireLock got %v, want ErrBusy", err)
	}
	if err := s.Update(func(v *State) { v.Workspace = "blocked" }); !errors.Is(err, ErrBusy) {
		release()
		t.Fatalf("Update while locked got %v, want ErrBusy", err)
	}
	afterRaw, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(afterRaw, raw) {
		release()
		t.Fatalf("lock diagnostics altered while busy: %q vs %q (%v)", afterRaw, raw, err)
	}

	release()
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock not removed after release: %v", err)
	}
	if err := s.Update(func(v *State) { v.Workspace = "/ok" }); err != nil {
		t.Fatalf("Update after release failed: %v", err)
	}
}

func TestHelperRecoveryDeadPID(t *testing.T) {
	if os.Getenv("WHALEISLE_TEST_RECOVERY_EXIT") == "1" {
		os.Exit(0)
	}
}

func TestStaleDeadPIDLockIsReclaimedWhileLiveOwnerStaysBusy(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestHelperRecoveryDeadPID$")
	cmd.Env = append(os.Environ(), "WHALEISLE_TEST_RECOVERY_EXIT=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadPID := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	lockPath := filepath.Join(dir, FileName+".lock")
	stalePayload, err := json.Marshal(lockOwnerInfo{
		PID:       deadPID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, append(stalePayload, '\n'), 0600); err != nil {
		t.Fatal(err)
	}

	// Dead PID lock must be automatically reclaimed so Update succeeds.
	if err := s.Update(func(v *State) { v.Workspace = "/reclaimed" }); err != nil {
		t.Fatalf("Update failed to reclaim dead-PID lock (pid=%d): %v", deadPID, err)
	}
	if got := s.Snapshot().Workspace; got != "/reclaimed" {
		t.Fatalf("workspace = %q, want /reclaimed", got)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file lingered after Update: %v", err)
	}

	// Live owner PID must still be rejected with ErrBusy.
	livePayload, err := json.Marshal(lockOwnerInfo{
		PID:       os.Getpid(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, append(livePayload, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(v *State) { v.Workspace = "/blocked" }); !errors.Is(err, ErrBusy) {
		t.Fatalf("live-PID lock got %v, want ErrBusy", err)
	}
}

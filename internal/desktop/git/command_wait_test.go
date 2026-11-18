package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestPipeHoldingDescendant is the subprocess helper for
// TestRunnerWaitCannotOutliveContext. The parent role starts a descendant that
// inherits its output pipe and then exits at once; the holder role keeps that
// pipe open without writing anything, and records its pid so the test can stop
// it instead of leaving it to the timeout.
func TestPipeHoldingDescendant(t *testing.T) {
	switch os.Getenv("WHALEISLE_COMMAND_PIPE_ROLE") {
	case "holder":
		time.Sleep(30 * time.Second)
		return
	case "parent":
		holder := exec.Command(os.Args[0], "-test.run=TestPipeHoldingDescendant")
		holder.Env = append(os.Environ(), "WHALEISLE_COMMAND_PIPE_ROLE=holder")
		holder.Stdout, holder.Stderr = os.Stdout, os.Stderr
		if err := holder.Start(); err != nil {
			t.Fatalf("descendant: %v", err)
		}
		pidFile := os.Getenv("WHALEISLE_COMMAND_PIPE_PID_FILE")
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(holder.Process.Pid)), 0o600)
		return
	}
	t.Skip("command runner subprocess helper")
}

func stopPipeHolder(pidFile string) {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return
	}
	if process, err := os.FindProcess(pid); err == nil {
		_ = process.Kill()
	}
}

// A descendant holding the output pipe must not keep the wait blocked past the
// context: a real git hook, credential helper or editor that outlives the child
// hangs the same way, and the user has no timeout to see.
func TestRunnerWaitCannotOutliveContext(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(t.TempDir(), "holder.pid")
	t.Setenv("WHALEISLE_COMMAND_PIPE_ROLE", "parent")
	t.Setenv("WHALEISLE_COMMAND_PIPE_PID_FILE", pidFile)
	defer stopPipeHolder(pidFile)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		_, _ = combinedExecRunner(ctx, "", executable, "-test.run=TestPipeHoldingDescendant")
	}()
	select {
	case <-returned:
		if elapsed := time.Since(started); elapsed > 6*time.Second {
			t.Fatalf("runner returned after %s, but its context expired after 1s", elapsed.Round(time.Millisecond))
		}
	case <-time.After(20 * time.Second):
		t.Fatal("runner never returned: the wait outlived its context")
	}
}

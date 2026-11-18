package processutil

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
// TestCommandContextWaitCannotOutliveContext. The parent role starts a
// descendant that inherits its output pipe and exits at once; the holder role
// keeps that pipe open without writing, and records its pid so the test can
// stop it rather than leaving it to the timeout.
func TestPipeHoldingDescendant(t *testing.T) {
	switch os.Getenv("WHALEISLE_PIPE_ROLE") {
	case "holder":
		time.Sleep(30 * time.Second)
		return
	case "parent":
		holder := exec.Command(os.Args[0], "-test.run=TestPipeHoldingDescendant")
		holder.Env = append(os.Environ(), "WHALEISLE_PIPE_ROLE=holder")
		holder.Stdout, holder.Stderr = os.Stdout, os.Stderr
		if err := holder.Start(); err != nil {
			t.Fatalf("descendant: %v", err)
		}
		_ = os.WriteFile(os.Getenv("WHALEISLE_PIPE_PID_FILE"), []byte(strconv.Itoa(holder.Process.Pid)), 0o600)
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

// Cancelling the context must end the wait for output even when a descendant
// still holds the pipe. Without WaitDelay this read never returns: no error, no
// timeout, and nothing the caller can do about it.
func TestCommandContextWaitCannotOutliveContext(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(t.TempDir(), "holder.pid")
	t.Setenv("WHALEISLE_PIPE_ROLE", "parent")
	t.Setenv("WHALEISLE_PIPE_PID_FILE", pidFile)
	defer stopPipeHolder(pidFile)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		_, _ = CommandContext(ctx, executable, "-test.run=TestPipeHoldingDescendant").CombinedOutput()
	}()
	select {
	case <-returned:
		if elapsed := time.Since(started); elapsed > 6*time.Second {
			t.Fatalf("output returned after %s, but the context expired after 1s", elapsed.Round(time.Millisecond))
		}
	case <-time.After(20 * time.Second):
		t.Fatal("output never returned: the wait outlived its context")
	}
}

// The grace must not disturb an ordinary command: it still completes and its
// output still arrives.
func TestCommandContextReturnsOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	out, err := CommandContext(ctx, executable, "-test.run=^$").CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Fatal("command produced no output")
	}
}

package pluginmanager

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func eventFixture(t *testing.T, phase string) (*Controller, uint64) {
	t.Helper()
	c := New()
	c.Attach(basicFixture(nil), nil)
	owner := c.Snapshot().Owner
	c.Actions(owner).edit(func(s *State) { s.Install = InstallState{RequestID: "request", Phase: phase} })
	return c, owner
}

func TestInstallProgressCannotReviveTerminalOrReopenCancellation(t *testing.T) {
	for _, phase := range []string{"idle", "checking", "done", "failed", "cancelled", "unknown", "reviewed"} {
		c, owner := eventFixture(t, phase)
		c.Progress(owner, InstallProgress{RequestID: "request", Phase: "installing"})
		if c.Snapshot().Install.Phase != phase {
			t.Fatalf("progress revived %s", phase)
		}
	}
	c, owner := eventFixture(t, "starting")
	c.Progress(owner, InstallProgress{Phase: "applying"})
	c.Progress(owner+1, InstallProgress{RequestID: "request", Phase: "applying"})
	c.Progress(owner, InstallProgress{RequestID: "other", Phase: "applying"})
	if c.Snapshot().Install.Phase != "starting" {
		t.Fatal("unowned/empty progress changed request")
	}
	registry := "https://fixture.invalid/registry"
	c.Progress(owner, InstallProgress{RequestID: "request", Phase: "installing", Attempt: &InstallAttempt{Registry: &registry, Index: 1, Total: 2}})
	i := c.Snapshot().Install
	if !i.Acknowledged || i.Phase != "running" || len(i.Attempts) != 1 || *i.Attempts[0].Registry != registry {
		t.Fatalf("start acknowledgement/attempt missing: %+v", i)
	}
	c.Progress(owner, InstallProgress{RequestID: "request", Phase: "cancelling"})
	c.Progress(owner, InstallProgress{RequestID: "request", Phase: "installing"})
	if c.Snapshot().Install.Phase != "cancelling" {
		t.Fatal("installing reopened cancellation")
	}
	c.Progress(owner, InstallProgress{RequestID: "request", Phase: "applying"})
	c.Progress(owner, InstallProgress{RequestID: "request", Phase: "cancelling"})
	if c.Snapshot().Install.Phase != "applying" {
		t.Fatal("late progress reopened applying")
	}
}

func TestInstallOutputAcknowledgesAndRetainsFinalExistingRun(t *testing.T) {
	c, owner := eventFixture(t, "starting")
	c.AppendLog(owner, InstallLog{RequestID: "request", JobID: "pnpm-1", Argv: []string{"pnpm", "add", "fixture"}, CWD: "fixture-profile", Stream: "stdout", Text: "starting\n"})
	i := c.Snapshot().Install
	if !i.Acknowledged || i.Phase != "running" || len(i.Runs) != 1 {
		t.Fatalf("output did not acknowledge pending start: %+v", i)
	}
	c.Actions(owner).edit(func(s *State) { s.Install.Phase = "done" })
	var final InstallLog
	if err := json.Unmarshal([]byte(`{"requestId":"request","jobId":"pnpm-1","argv":["ignored"],"cwd":"ignored","stream":"stderr","text":"final\n","exitCode":null}`), &final); err != nil {
		t.Fatal(err)
	}
	c.AppendLog(owner, final)
	c.AppendLog(owner, InstallLog{RequestID: "request", JobID: "late-new-run", Text: "must ignore"})
	c.AppendLog(owner+1, InstallLog{RequestID: "request", JobID: "pnpm-1", Text: "wrong owner"})
	i = c.Snapshot().Install
	run := i.Runs[0]
	if i.Phase != "done" || len(i.Runs) != 1 || run.Command != "pnpm add fixture" || run.CWD != "fixture-profile" || run.Output != "starting\nfinal\n" || len(run.Chunks) != 2 || run.Chunks[1].Stream != "stderr" || !run.Settled || run.ExitCode != nil {
		t.Fatalf("final output lost original command/stream/null exit: %+v", run)
	}
	exit := 7
	c.AppendLog(owner, InstallLog{RequestID: "request", JobID: "pnpm-1", ExitCode: &exit})
	if *c.Snapshot().Install.Runs[0].ExitCode != 7 {
		t.Fatal("numeric exit code missing")
	}
	c.AppendLog(owner, InstallLog{RequestID: "request", JobID: "pnpm-1", Stream: "stdout", Text: strings.Repeat("x", 70000)})
	i = c.Snapshot().Install
	if len(i.Log) > 64000 || len(i.Runs[0].Output) > 64000 || !i.Runs[0].Truncated || i.Runs[0].Command != "pnpm add fixture" {
		t.Fatal("output limit lost metadata or failed to bound")
	}
}

func TestInstallOutputRetriesCancellationThatOvertookStart(t *testing.T) {
	c, owner := eventFixture(t, "starting")
	base := basicFixture(nil)
	requests := make(chan string, 2)
	c.mu.Lock()
	c.client = fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if method == "pluginManager/cancelInstall" {
			requests <- method
			return reply(out, map[string]string{"status": "not-running"})
		}
		return base(ctx, method, args, out)
	})
	c.mu.Unlock()
	a := c.Actions(owner)
	a.CancelInstall()
	waitState(t, c, func(s State) bool { return s.Install.WaitingForStart })
	<-requests
	c.AppendLog(owner, InstallLog{RequestID: "request", JobID: "run", Stream: "stdout", Text: "accepted"})
	waitState(t, c, func(s State) bool { return s.Install.Acknowledged && !s.Install.CancellationRequested })
	select {
	case <-requests:
	default:
		t.Fatal("acknowledged output did not retry pending cancellation")
	}
	if c.Snapshot().Install.Phase == "running" {
		t.Fatal("output reopened cancelled window")
	}
}

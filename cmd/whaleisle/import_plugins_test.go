package main

import (
	"context"
	"reflect"
	"testing"

	"whaleisle/internal/harness/installctrl"
)

func TestImportPluginRequiresExactBuildApproval(t *testing.T) {
	var calls [][]string
	approvals := 0
	out := installImportPlugin(context.Background(), "fixture@1.0.0", func(allow []string) installctrl.Result {
		calls = append(calls, append([]string(nil), allow...))
		if len(allow) == 0 { return installctrl.Result{NeedsAllowBuilds: true, AllowBuilds: []string{"esbuild"}, Log: "first"} }
		return installctrl.Result{OK: true, Log: "second"}
	}, func(builds []string) bool {
		approvals++
		if !reflect.DeepEqual(builds, []string{"esbuild"}) { t.Fatalf("approval widened: %v", builds) }
		return true
	})
	if !out.OK || approvals != 1 || len(calls) != 2 || len(calls[0]) != 0 || !reflect.DeepEqual(calls[1], []string{"esbuild"}) || out.Log != "first\nsecond" { t.Fatalf("approval flow: %+v calls=%v approvals=%d", out, calls, approvals) }
}

func TestImportPluginDeclinedOrCancelledApprovalNeverRetries(t *testing.T) {
	for _, cancelOnApprove := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		out := installImportPlugin(ctx, "fixture@1.0.0", func([]string) installctrl.Result {
			calls++
			return installctrl.Result{NeedsAllowBuilds: true, AllowBuilds: []string{"esbuild"}}
		}, func([]string) bool {
			if cancelOnApprove { cancel(); return true }
			return false
		})
		cancel()
		if out.OK || calls != 1 { t.Fatalf("retry after refusal/cancel: %+v calls=%d", out, calls) }
	}
}

func TestImportPluginRepeatedBuildFailureDoesNotLoop(t *testing.T) {
	calls, approvals := 0, 0
	out := installImportPlugin(context.Background(), "fixture@1.0.0", func([]string) installctrl.Result {
		calls++
		if calls > 2 { t.Fatal("unbounded retry") }
		return installctrl.Result{NeedsAllowBuilds: true, AllowBuilds: []string{"esbuild"}}
	}, func([]string) bool { approvals++; return true })
	if out.OK || calls != 2 || approvals != 1 { t.Fatalf("repeated build: %+v calls=%d approvals=%d", out, calls, approvals) }
}

func TestMarketAdmissionBlockedDuringImport(t *testing.T) {
	s := &mainState{launcherImportBusy: true}
	if op, _ := s.beginMarketAttempt("github:fixture/plugin", nil, true); op != nil {
		t.Fatal("market installation admitted during import")
	}
}

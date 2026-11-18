package main

import (
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

// TestConnectionFailureSentenceNamesTheRuntimeExit pins the boot dock's copy:
// when the Host's process is gone, the supervisor's error is the reason a person
// needs, and the mux error that surfaced it is only a symptom. Falling through to
// the generic sentence hides both the cause and the exit code.
func TestConnectionFailureSentenceNamesTheRuntimeExit(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	failed := harness.Status{State: harness.Failed, Err: errors.New("dsh web exited: exit status 1")}
	for _, muxErr := range []error{
		errors.New("dsh mux: carrier lost: websocket: close 1006"),
		errors.New("Host bootstrap unavailable"),
		errors.New("revision mismatch"),
	} {
		got := connectionFailureSentence(muxErr, failed)
		if !strings.Contains(got, "桌面运行时已退出") {
			t.Fatalf("mux error %q = %q, want the runtime-exit sentence", muxErr, got)
		}
		if !strings.Contains(got, "退出码 1") {
			t.Fatalf("mux error %q = %q, want the exit code the supervisor reported", muxErr, got)
		}
		if strings.Contains(got, "操作没有完成") || strings.Contains(got, "mux") {
			t.Fatalf("mux error %q = %q fell through to the generic sentence", muxErr, got)
		}
	}
}

// TestConnectionFailureSentenceKeepsTheLiveHostCause keeps the supervisor from
// speaking for a Host that is running: a network failure while the runtime is up
// is the network's own sentence.
func TestConnectionFailureSentenceKeepsTheLiveHostCause(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	ready := harness.Status{State: harness.Ready}
	got := connectionFailureSentence(errors.New("dial tcp 127.0.0.1:3080: connect: connection refused"), ready)
	if !strings.Contains(got, "网络") {
		t.Fatalf("live Host network failure = %q", got)
	}
	// A failed Host whose supervisor kept no error still names its own failure.
	unknown := connectionFailureSentence(errors.New("revision mismatch"), harness.Status{State: harness.Failed})
	if !strings.Contains(unknown, "操作没有完成") {
		t.Fatalf("codeless failure = %q", unknown)
	}
}

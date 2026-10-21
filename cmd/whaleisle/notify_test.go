package main

import (
	"errors"
	"strings"
	"testing"
)

// TestNotifyUserShowsOnce pins the queue: a queued message is painted by exactly
// one frame, so a toast never repeats while nothing else happens.
func TestNotifyUserShowsOnce(t *testing.T) {
	s := &mainState{}
	s.notifyUser(nil, "尚未连接 Host，请稍后重试")
	if got := s.takePendingToast(); got != "尚未连接 Host，请稍后重试" {
		t.Fatalf("first read = %q", got)
	}
	if got := s.takePendingToast(); got != "" {
		t.Fatalf("toast repeated: %q", got)
	}
}

// TestNotifyUserIgnoresEmptyCopy keeps silent no-ops from painting blank toasts.
func TestNotifyUserIgnoresEmptyCopy(t *testing.T) {
	s := &mainState{}
	s.notifyUser(nil, "   ")
	if got := s.takePendingToast(); got != "" {
		t.Fatalf("blank toast queued: %q", got)
	}
}

// TestSaveDesktopConfigAnswersWhenTheHostIsMissing covers the audited silent
// return: saving desktop settings without a configuration store must answer the
// click instead of doing nothing.
func TestSaveDesktopConfigAnswersWhenTheHostIsMissing(t *testing.T) {
	s := &mainState{}
	s.saveDesktopConfig()
	if got := s.takePendingToast(); got == "" || !strings.Contains(got, "Host") {
		t.Fatalf("no visible refusal: %q", got)
	}
}

// TestSaveFailedToastKeepsTheHostReason makes sure failures name the action and
// keep the Host's wording instead of swallowing the error.
func TestSaveFailedToastKeepsTheHostReason(t *testing.T) {
	got := saveFailedToast("保存标题", errors.New("revision mismatch"))
	if !strings.Contains(got, "保存标题") || !strings.Contains(got, "revision mismatch") {
		t.Fatalf("failure copy = %q", got)
	}
	if bare := saveFailedToast("保存标题", nil); !strings.Contains(bare, "失败") {
		t.Fatalf("bare failure copy = %q", bare)
	}
}

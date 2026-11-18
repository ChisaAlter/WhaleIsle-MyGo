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

// TestSaveFailedToastNamesTheCauseWithoutTheHostReason pins the copy rule: the
// action and an actionable cause reach the toast, the Host's own wording does not.
func TestSaveFailedToastNamesTheCauseWithoutTheHostReason(t *testing.T) {
	zhForTest(t)
	got := saveFailedToast("保存标题", errors.New(`open C:\notes.md: Access is denied.`))
	if !strings.Contains(got, "保存标题失败") {
		t.Fatalf("toast does not name the action: %q", got)
	}
	if !strings.Contains(got, "权限") {
		t.Fatalf("toast does not explain the cause: %q", got)
	}
	for _, raw := range []string{"Access is denied", `C:\notes.md`} {
		if strings.Contains(got, raw) {
			t.Fatalf("toast leaked %q: %q", raw, got)
		}
	}
	if bare := saveFailedToast("保存标题", nil); bare != "保存标题失败" {
		t.Fatalf("bare failure copy = %q", bare)
	}
}

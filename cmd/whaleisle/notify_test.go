package main

import (
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

// TestNotifyUserWritesTheStatusLine pins the surface: an answer to a click lands
// in the conversation's own status line, never in a toast over the content.
func TestNotifyUserWritesTheStatusLine(t *testing.T) {
	store := session.New()
	s := &mainState{sessions: store}
	s.notifyUser(nil, "尚未连接 Host，请稍后重试")
	if got := store.Snapshot().LastAction; got != "尚未连接 Host，请稍后重试" {
		t.Fatalf("status line = %q", got)
	}
}

// TestNotifyUserBeforeTheShellKeepsTheLineInTheLog covers the window before the
// session store exists: the message joins the startup log instead of vanishing.
func TestNotifyUserBeforeTheShellKeepsTheLineInTheLog(t *testing.T) {
	s := &mainState{}
	s.notifyUser(nil, "尚未连接 Host，请稍后重试")
	if len(s.bootLogs) != 1 || !strings.Contains(s.bootLogs[0], "Host") {
		t.Fatalf("startup log = %v", s.bootLogs)
	}
}

// TestNotifyUserIgnoresEmptyCopy keeps silent no-ops from writing a blank line.
func TestNotifyUserIgnoresEmptyCopy(t *testing.T) {
	store := session.New()
	s := &mainState{sessions: store}
	s.notifyUser(nil, "   ")
	if got := store.Snapshot().LastAction; got != "" {
		t.Fatalf("blank line written: %q", got)
	}
}

// TestSaveDesktopConfigAnswersWhenTheHostIsMissing covers the audited silent
// return: saving desktop settings without a configuration store must answer the
// click instead of doing nothing.
func TestSaveDesktopConfigAnswersWhenTheHostIsMissing(t *testing.T) {
	store := session.New()
	s := &mainState{sessions: store}
	s.saveDesktopConfig()
	if got := store.Snapshot().LastAction; got == "" || !strings.Contains(got, "Host") {
		t.Fatalf("no visible refusal: %q", got)
	}
}

// TestSaveFailedToastNamesTheCauseWithoutTheHostReason pins the copy rule: the
// action and an actionable cause reach the line a person reads, the Host's own
// wording does not.
func TestSaveFailedToastNamesTheCauseWithoutTheHostReason(t *testing.T) {
	zhForTest(t)
	got := saveFailedToast("保存标题", errors.New(`open C:\notes.md: Access is denied.`))
	if !strings.Contains(got, "保存标题失败") {
		t.Fatalf("line does not name the action: %q", got)
	}
	if !strings.Contains(got, "权限") {
		t.Fatalf("line does not explain the cause: %q", got)
	}
	for _, raw := range []string{"Access is denied", `C:\notes.md`} {
		if strings.Contains(got, raw) {
			t.Fatalf("line leaked %q: %q", raw, got)
		}
	}
	if bare := saveFailedToast("保存标题", nil); bare != "保存标题失败" {
		t.Fatalf("bare failure copy = %q", bare)
	}
}

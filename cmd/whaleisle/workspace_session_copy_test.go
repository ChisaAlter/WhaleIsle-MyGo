package main

import (
	"strings"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestNewSessionWithoutAWorkspaceAsksForAFolder covers AYA-33's copy half: with
// no workspace at all, pressing 新会话 used to answer "the workspace list is not
// ready, wait for the connection to come back". Nothing is reconnecting in that
// state; the next step is choosing a folder.
func TestNewSessionWithoutAWorkspaceAsksForAFolder(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	store := session.New()
	s := &mainState{}
	s.openGlobalWorkspaceSession(nil, store)

	action := store.Snapshot().LastAction
	if action == "" {
		t.Fatal("no answer to 新会话")
	}
	if !strings.Contains(action, "选择") || !strings.Contains(action, "文件夹") {
		t.Fatalf("the answer does not say to pick a folder: %q", action)
	}
	if strings.Contains(action, "尚未就绪") || strings.Contains(action, "连接恢复") {
		t.Fatalf("an actionable state is still described as a wait: %q", action)
	}
}

// TestNewSessionWaitsOnlyWhenAWorkspaceIsRecorded keeps the waiting sentence for
// the state it is true in: a folder is chosen and the Host has not caught up.
func TestNewSessionWaitsOnlyWhenAWorkspaceIsRecorded(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	store := session.New()
	store.SetWorkspaceCwd(`C:\work\project`)
	s := &mainState{}
	s.openGlobalWorkspaceSession(nil, store)

	action := store.Snapshot().LastAction
	if !strings.Contains(action, "尚未就绪") {
		t.Fatalf("a recorded workspace lost the waiting sentence: %q", action)
	}
}

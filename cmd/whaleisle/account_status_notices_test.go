package main

import (
	"context"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

// A signed-out account is a state the conversation's status line carries, and it
// is written once per notice: the line never repeats on every painted frame.
func TestAccountStatusNoticeWritesTheStatusLineOnce(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	client := &dsh.Client{}
	store := usage.New()
	store.Attach(client, nil)
	sessions := session.New()
	s := &mainState{usage: store, sessions: sessions}

	store.OnAccountNotice(context.Background(), client, dsh.EventsEmit{Event: "deepseek-account/session-expired"})
	s.syncAccountStatusNotices()
	want := i18n.T(i18n.KeyCmdYouHaveSignedOutOfYourAccountPleaseL)
	if got := sessions.Snapshot().LastAction; got != want {
		t.Fatalf("status line = %q, want %q", got, want)
	}

	sessions.SetLastAction("another action")
	s.syncAccountStatusNotices()
	if got := sessions.Snapshot().LastAction; got != "another action" {
		t.Fatalf("the same notice was written again: %q", got)
	}
}

// A model that needs a sign-in is the same kind of state, with its own sentence.
func TestAccountStatusNoticeNamesTheModelSignIn(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })
	client := &dsh.Client{}
	store := usage.New()
	store.Attach(client, nil)
	sessions := session.New()
	s := &mainState{usage: store, sessions: sessions}

	store.OnAccountNotice(context.Background(), client, dsh.EventsEmit{Event: "deepseek-account/model-sign-in-required"})
	s.syncAccountStatusNotices()
	want := i18n.T(i18n.KeyCmdModelUnavailablePleaseSignInAndTryAga)
	if got := sessions.Snapshot().LastAction; got != want {
		t.Fatalf("status line = %q, want %q", got, want)
	}
}

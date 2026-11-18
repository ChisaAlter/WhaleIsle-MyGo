package main

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/sidebar"
)

// Replacing the session store releases the frame memory the retired store owned
// and leaves the new store ready to be remembered on its first paint.
func TestReplaceSessionsForgetsRetiredStoreBookkeeping(t *testing.T) {
	state := &mainState{}
	retired := session.New()
	state.sessions = retired
	frame.RememberExternalAccount(retired, sidebar.Account{Name: "retired"})

	next := session.New()
	t.Cleanup(func() { frame.ForgetExternalAccount(next) })
	state.replaceSessionsLocked(next)

	if state.sessions != next {
		t.Fatal("replaceSessionsLocked did not install the new store")
	}
	if _, ok := frame.RememberedExternalAccount(retired); ok {
		t.Fatal("retired store is still remembered by the frame")
	}
	frame.RememberExternalAccount(next, sidebar.Account{Name: "next"})
	if account, ok := frame.RememberedExternalAccount(next); !ok || account.Name != "next" {
		t.Fatalf("new store was not remembered: %#v, %v", account, ok)
	}
}

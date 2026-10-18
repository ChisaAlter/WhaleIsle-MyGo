package main

import (
	"testing"
	"whaleisle/internal/store/usage"
)

func TestAccountStatusNoticeLoginActionOwnership(t *testing.T) {
	for _, change := range []string{"current", "store", "account", "host", "quitting"} {
		t.Run(change, func(t *testing.T) {
			store := usage.New()
			s := &mainState{usage: store}
			toast := s.accountStatusToast(store, store.Notices(), "expired", "Sign in again")
			if toast.Action == "" || toast.OnAction == nil {
				t.Fatal("login action absent")
			}
			switch change {
			case "store":
				s.usage = usage.New()
			case "account":
				store.Attach(nil, nil)
			case "host":
				s.bootStatus.Generation++
			case "quitting":
				s.quitting = true
			}
			toast.OnAction()
			if s.settingsOpen != (change == "current") || s.settingsAccountActive != (change == "current") {
				t.Fatalf("unexpected account navigation: %v %v", s.settingsOpen, s.settingsAccountActive)
			}
		})
	}
}

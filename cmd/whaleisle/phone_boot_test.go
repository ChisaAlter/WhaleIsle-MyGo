package main

import (
	"testing"

	"whaleisle/internal/config"
)

// newPhoneBootState builds the smallest mainState that can decide the boot
// start: a temp config store, a temp user data dir and a counted start action
// instead of a real daemon.
func newPhoneBootState(t *testing.T, remoteEnabled bool, quitting bool) (*mainState, *int) {
	t.Helper()
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if remoteEnabled {
		if err := store.Update(func(c *config.Config) { c.RemoteEnabled = true }); err != nil {
			t.Fatal(err)
		}
	}
	starts := 0
	s := &mainState{
		cfgStore:             store,
		quitting:             quitting,
		phoneBootStartAction: func() { starts++ },
	}
	return s, &starts
}

// The boot start is decided by claimPhoneBootStartLocked, which refreshBoot
// calls on every Ready transition while holding s.mu. These cases mirror the
// required behaviour: enabled+ready once, repeated ready frames never restart,
// disabled never, quitting skips. (refreshBoot itself cannot be driven in a
// unit test: its tail dispatches through mygo.RunOnMain, which waits for the
// toolkit event loop that unit tests do not run.)
func TestClaimPhoneBootStartLocked(t *testing.T) {
	for _, tc := range []struct {
		name     string
		enabled  bool
		quitting bool
		want     []bool
	}{
		{name: "enabled starts once", enabled: true, want: []bool{true, false, false}},
		{name: "repeated ready frames never restart", enabled: true, want: []bool{true, false}},
		{name: "disabled never starts", want: []bool{false, false, false}},
		{name: "quitting skips", enabled: true, quitting: true, want: []bool{false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, starts := newPhoneBootState(t, tc.enabled, tc.quitting)
			for i, want := range tc.want {
				s.mu.Lock()
				got := s.claimPhoneBootStartLocked()
				s.mu.Unlock()
				if got != want {
					t.Fatalf("call %d = %v, want %v", i, got, want)
				}
				if *starts != 0 {
					t.Fatalf("decision ran the start action directly (%d)", *starts)
				}
			}
			s.mu.Lock()
			started := s.phoneBootStarted
			s.mu.Unlock()
			if tc.enabled && !tc.quitting && !started {
				t.Fatal("enabled boot did not mark the gate started")
			}
			if (!tc.enabled || tc.quitting) && started {
				t.Fatal("disabled/quitting boot consumed the gate")
			}
		})
	}
}

// The action runs after the lock is released, mirroring refreshBoot's order; a
// nil action means the real refreshPhoneRemote in production.
func TestPhoneBootStartActionRunsOutsideTheGate(t *testing.T) {
	s, starts := newPhoneBootState(t, true, false)
	s.mu.Lock()
	claim := s.claimPhoneBootStartLocked()
	s.mu.Unlock()
	if !claim {
		t.Fatal("first ready did not claim the boot start")
	}
	if s.phoneBootStartAction != nil {
		s.phoneBootStartAction()
	}
	if *starts != 1 {
		t.Fatalf("action calls = %d, want 1", *starts)
	}
	// A second Ready frame claims nothing, so the action is never issued again.
	s.mu.Lock()
	again := s.claimPhoneBootStartLocked()
	s.mu.Unlock()
	if again {
		t.Fatal("second ready claimed the boot start again")
	}
}

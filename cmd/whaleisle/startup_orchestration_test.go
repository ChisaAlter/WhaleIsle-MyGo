package main

import (
	"testing"
	"time"

	"whaleisle/internal/app/startup"
	"whaleisle/internal/app/updater"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Tier C subsystems registered at window creation stay unstarted until their
// feature is used (or the idle window passes after WorkbenchReady).
func TestTierCSubsystemsWaitForTheirTrigger(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	t.Setenv(updater.EnvDisableAutoUpdate, "1")
	s := &mainState{}
	s.registerDeferredStartup(nil, t.TempDir(), false)
	s.deferPhoneBootStart(func() {})
	_, plan := s.startupPlan()
	gen := s.beginStartupGeneration()
	tracker, _ := s.startupPlan()
	tracker.Reach(gen, startup.EventsReady)
	tracker.Reach(gen, startup.WorkbenchReady)
	time.Sleep(20 * time.Millisecond)
	for _, name := range []string{"update.center", "proactive.monitor", "im.store", "phone.gateway"} {
		if n := plan.Started(name); n != 0 {
			t.Fatalf("%s started %d times before its trigger", name, n)
		}
	}
	s.noteRouteUse(routeUpdatePreferences)
	if plan.Started("update.center") != 1 {
		t.Fatal("opening the update page must start the update center")
	}
	for _, name := range []string{"proactive.monitor", "im.store", "phone.gateway"} {
		if plan.Started(name) != 0 {
			t.Fatalf("%s started on another feature's first use", name)
		}
	}
	s.noteRouteUse(routeRemote)
	if plan.Started("phone.gateway") != 1 || plan.Started("ssh.tunnel") != 1 {
		t.Fatal("the remote page starts the phone gateway and the cold stores holding the SSH tunnel")
	}
	s.noteRouteUse(routeUpdatePreferences)
	if plan.Started("update.center") != 1 {
		t.Fatal("a Tier C group starts once per process")
	}
}

func TestBootProgressOnlyIgnoresLogAndPluginProgress(t *testing.T) {
	cur := harness.Status{State: harness.Starting, Generation: 3, Attempt: 1}
	next := cur
	next.PluginLoad = harness.PluginProgress{Phase: "plugins", Ready: 4, Total: 9}
	if !bootProgressOnly(cur, next) {
		t.Fatal("plugin progress inside one Starting status is progress only")
	}
	for _, change := range []func(*harness.Status){
		func(s *harness.Status) { s.State = harness.Ready },
		func(s *harness.Status) { s.Generation++ },
		func(s *harness.Status) { s.RecoveryScheduled = true },
		func(s *harness.Status) { s.Attempt++ },
	} {
		n := cur
		change(&n)
		if bootProgressOnly(cur, n) {
			t.Fatalf("a state change was treated as progress: %+v", n)
		}
	}
}

func TestBootLogSnapshotIsImmutable(t *testing.T) {
	s := &mainState{}
	s.bootLogs.Publish([]string{"a", "b"})
	held := s.bootLogSnapshot()
	s.mu.Lock()
	s.appendBootLogLocked("c")
	s.mu.Unlock()
	if len(held) != 2 || len(s.bootLogSnapshot()) != 3 {
		t.Fatalf("held=%v now=%v", held, s.bootLogSnapshot())
	}
	_ = append(held, "x")
	if got := s.bootLogSnapshot(); got[2] != "c" {
		t.Fatalf("a reader's append changed the published log: %v", got)
	}
}

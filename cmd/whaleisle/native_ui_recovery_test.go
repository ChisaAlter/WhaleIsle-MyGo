package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func TestNativeUIRecoveryControllerPauseAndInFlightGuard(t *testing.T) {
	ctrl := newNativeUIRecoveryController()
	now := time.Now()
	if ctrl.shouldPauseLocked(now) {
		t.Fatal("newly armed recovery controller paused immediately")
	}
	ctrl.activeUntilNano.Store(now.Add(-time.Second).UnixNano())
	if !ctrl.shouldPauseLocked(now) {
		t.Fatal("idle recovery controller did not pause after inactivity window")
	}
	// An in-flight RunOnMain probe must never be masked as paused, so genuine
	// UI-thread hangs still escalate to the watchdog after nativeUIHangAfter.
	ctrl.inFlightBeat.Store(true)
	if ctrl.shouldPauseLocked(now) {
		t.Fatal("recovery controller paused while main-thread beat was in flight")
	}
	ctrl.inFlightBeat.Store(false)
	s := &mainState{}
	s.nativeRecovery.Store(ctrl)
	s.pauseNativeUIRecovery()
	if !ctrl.hidden.Load() || !ctrl.shouldPauseLocked(now) {
		t.Fatal("pauseNativeUIRecovery did not suppress idle heartbeats on hide/minimize")
	}
	s.resumeNativeUIRecovery()
	if ctrl.hidden.Load() || ctrl.shouldPauseLocked(time.Now()) {
		t.Fatal("resumeNativeUIRecovery did not re-arm active monitoring")
	}
}

// recordingRecoverySurface is the platform boundary the watchdog's window sits
// on. The recovery window cannot be created in a test without an event loop, so
// what is asserted here is the contract the real window is reached through.
type recordingRecoverySurface struct {
	inactiveShows int
}

func (r *recordingRecoverySurface) ShowInactive() { r.inactiveShows++ }

func TestRecoveryWindowIsCreatedWithoutActivation(t *testing.T) {
	opts := recoveryWindowOptions(lifecycleText("界面未响应", "Interface not responding"))
	if !opts.Hidden {
		// mygo raises a window created with Hidden false through Show(), which
		// calls SetForegroundWindow: a recovery surface for a hung desktop would
		// take the foreground from whatever the operator is working in.
		t.Fatal("the watchdog window is created visible, so creating it activates it")
	}
	// AlwaysOnTop is a deliberate choice, not a leftover: it is the only reason
	// the window is seen when the frozen desktop is the window in front. A
	// change here has to say why the window would still be noticed.
	if !opts.AlwaysOnTop {
		t.Fatal("the watchdog window no longer stays above the frozen desktop it reports")
	}
	if opts.Title == "" {
		t.Fatal("the watchdog window lost its title")
	}
}

func TestWatchdogRaisesItsWindowWithoutActivation(t *testing.T) {
	var _ recoverySurface = (*mygo.Window)(nil)
	surface := &recordingRecoverySurface{}
	raiseRecoverySurface(surface)
	if surface.inactiveShows != 1 {
		t.Fatalf("inactive shows = %d, want 1", surface.inactiveShows)
	}
	// Show and Focus are not on recoverySurface, so the raise path cannot take
	// the foreground by type; this test would not compile if one were added.
}

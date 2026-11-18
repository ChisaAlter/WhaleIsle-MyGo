package main

import (
	"testing"
	"time"
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

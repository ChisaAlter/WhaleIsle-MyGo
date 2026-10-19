//go:build windows

package ui

import (
	"testing"
	"time"
)

// CPU duty-budget assertions live in decorative_fluid_cpu_test.go. These
// platform contract checks intentionally neither allocate a GPU nor claim that
// a 64-cell CPU frame is evidence for the real Windows half-float simulation.
func TestSplashWindowsProfileAndCoalescedNativeInput(t *testing.T) {
	if !nativeSplashFits(1440, 900) || nativeSplashFits(16384, 16384) || nativeSplashFits(1, 1000000) {
		t.Fatal("Windows GPU allocation preflight changed")
	}
	if splashPacing(30*time.Millisecond) != splashFrameInterval {
		t.Fatal("Windows used the non-Windows CPU 24% duty limiter")
	}
	s := decorationState{config: splashTestConfig(), width: 200, height: 100}
	now := time.Unix(1000, 0)
	s.noteSplashPointer(100, 50, now)
	s.noteSplashPointer(104, 48, now)
	if s.fluidInput.dx < .019999 || s.fluidInput.dx > .020001 || s.fluidInput.dy < .009999 || s.fluidInput.dy > .010001 {
		t.Fatalf("not the latest aspect-corrected pointer delta: %+v", s.fluidInput)
	}
	s.pointerDown(80, 40)
	if len(s.fluidInput.presses) != 1 || s.fluidPointerX != .4 || s.fluidPointerY != .6 {
		t.Fatal("click failed to update the source pointer origin")
	}
	s.noteSplashPointer(82, 40, now)
	if s.fluidInput.dx < .009999 || s.fluidInput.dx > .010001 {
		t.Fatal("motion jumped from the pre-click pointer position")
	}
}

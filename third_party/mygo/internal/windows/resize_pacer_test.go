//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"
	"time"
)

func TestResizeIntervalForRefreshRate(t *testing.T) {
	if got := resizeIntervalForRefreshRate(0); got != 16*time.Millisecond {
		t.Fatalf("hz=0: got %v, want 16ms", got)
	}
	if got := resizeIntervalForRefreshRate(144); got != 16*time.Millisecond {
		t.Fatalf("hz=144: got %v, want clamped 16ms", got)
	}
	if got := resizeIntervalForRefreshRate(60); got < 16*time.Millisecond || got > 17*time.Millisecond {
		t.Fatalf("hz=60: got %v, want ~16.67ms", got)
	}
	if got := resizeIntervalForRefreshRate(30); got != 33*time.Millisecond {
		t.Fatalf("hz=30: got %v, want clamped 33ms", got)
	}
	if got := resizeIntervalForRefreshRate(10); got != 33*time.Millisecond {
		t.Fatalf("hz=10: got %v, want clamped 33ms", got)
	}
}

func TestResizePacerLiveThrottleAndFlush(t *testing.T) {
	var p resizePacer
	base := time.Unix(100, 0)
	interval := 16 * time.Millisecond

	// First WM_SIZE draws immediately (never freezes).
	drawNow, delay := p.onSize(base, interval)
	if !drawNow || delay != 0 {
		t.Fatalf("first resize: drawNow=%v delay=%v; want true, 0", drawNow, delay)
	}

	// Burst steps inside 16ms coalesce into a single timer arm.
	drawNow, delay = p.onSize(base.Add(4*time.Millisecond), interval)
	if drawNow || delay != 12*time.Millisecond {
		t.Fatalf("step 2: drawNow=%v delay=%v; want false, 12ms", drawNow, delay)
	}
	drawNow, delay = p.onSize(base.Add(9*time.Millisecond), interval)
	if drawNow || delay != 0 {
		t.Fatalf("step 3 (timer already armed): drawNow=%v delay=%v; want false, 0", drawNow, delay)
	}

	// Timer tick flushes the latest size in real time while still dragging.
	if !p.onTimer(base.Add(16 * time.Millisecond)) {
		t.Fatal("timer tick did not flush pending resize")
	}
	if p.onTimer(base.Add(17 * time.Millisecond)) {
		t.Fatal("spurious second timer tick flushed when nothing pending")
	}

	// Another resize right before release must flush immediately on WM_EXITSIZEMOVE.
	drawNow, delay = p.onSize(base.Add(20*time.Millisecond), interval)
	if drawNow || delay != 12*time.Millisecond {
		t.Fatalf("step 4: drawNow=%v delay=%v; want false, 12ms", drawNow, delay)
	}
	if !p.onExitSizeMove(base.Add(22 * time.Millisecond)) {
		t.Fatal("WM_EXITSIZEMOVE did not flush pending resize")
	}
	if p.onExitSizeMove(base.Add(23 * time.Millisecond)) {
		t.Fatal("second WM_EXITSIZEMOVE should be a no-op")
	}

	// An external WM_PAINT satisfies a pending resize and asks to cancel the timer.
	drawNow, delay = p.onSize(base.Add(25*time.Millisecond), interval)
	if drawNow || delay == 0 {
		t.Fatalf("step 5: drawNow=%v delay=%v; want false, >0", drawNow, delay)
	}
	if !p.onPainted(base.Add(28 * time.Millisecond)) {
		t.Fatal("onPainted should report active timer to cancel")
	}
	if p.pending || p.timerOn {
		t.Fatalf("state not cleared after paint: %+v", p)
	}
}

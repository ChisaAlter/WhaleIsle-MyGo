package ui

import (
	"errors"
	"testing"
	"time"
)

func TestSplashQueueCapsAndClearsOnErrorOrOcclusion(t *testing.T) {
	tt := NewTester(func(c *Context) {}, 200, 100)
	defer tt.rt.close()

	s := &tt.rt.decorations
	s.config = splashTestConfig()
	s.width, s.height = 200, 100

	for i := 0; i < maxQueuedSplashPresses+16; i++ {
		s.pointerDown(float32(i), 10)
	}
	if len(s.fluidInput.presses) != maxQueuedSplashPresses {
		t.Fatalf("press queue length = %d, want %d", len(s.fluidInput.presses), maxQueuedSplashPresses)
	}
	wantFirstX := float32(16) / 200
	if got := s.fluidInput.presses[0].x; got != wantFirstX {
		t.Fatalf("oldest retained press x = %v, want %v (FIFO drop of oldest)", got, wantFirstX)
	}

	now := time.Unix(2000, 0)
	tt.h.hidden = true
	s.paintSplash(&tt.rt.painter, 200, 100, now)
	if len(s.fluidInput.presses) != 0 || s.pendingPress {
		t.Fatalf("occluded paintSplash did not clear pending input: presses=%d pending=%v", len(s.fluidInput.presses), s.pendingPress)
	}

	tt.h.hidden = false
	s.fluidError = errors.New("gpu init failed")
	s.pointerDown(50, 50)
	if len(s.fluidInput.presses) != 0 {
		t.Fatalf("pointerDown queued press after fluidError: len=%d", len(s.fluidInput.presses))
	}
	beforeLast := s.fluidLastInput
	s.queueSplash(50, 50, 10, 10, now.Add(time.Second))
	if s.fluidInput.moved || !s.fluidLastInput.Equal(beforeLast) {
		t.Fatalf("queueSplash mutated fluid state after fluidError")
	}
	s.paintSplash(&tt.rt.painter, 200, 100, now.Add(2*time.Second))
	if s.pendingPress || len(s.fluidInput.presses) != 0 {
		t.Fatalf("paintSplash did not clear press state after fluidError")
	}
}

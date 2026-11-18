package startupsuggest

import (
	"testing"
	"time"
)

// run feeds frames every 16ms of a dt-long stretch, each a build of the given
// cost, and reports the frame indices that asked for a suggestion.
func run(o *Observer, start time.Time, dt time.Duration, build time.Duration, foreground bool) (time.Time, []time.Duration) {
	var hits []time.Duration
	for at := time.Duration(0); at < dt; at += 16 * time.Millisecond {
		now := start.Add(at)
		if o.Observe(now, build, foreground) {
			hits = append(hits, at)
		}
	}
	return start.Add(dt), hits
}

func TestSlowStartAsksOnce(t *testing.T) {
	o := New()
	now, hits := run(o, time.Now(), ReadyWindow, 16*time.Millisecond, true)
	if len(hits) != 0 {
		t.Fatalf("the ready window asked for a suggestion: %v", hits)
	}
	// The frame that closes the ready window; then stalls in two separate spans
	// of the observation, with ordinary frames in between, as a genuinely
	// stuttering interface shows them.
	now = now.Add(16 * time.Millisecond)
	o.Observe(now, 4*time.Millisecond, true)
	for i := 0; i < 2; i++ {
		now = now.Add(16 * time.Millisecond)
		o.Observe(now, 120*time.Millisecond, true)
	}
	now, _ = run(o, now, StallSpan, 4*time.Millisecond, true)
	if !o.Observe(now, 90*time.Millisecond, true) {
		t.Fatal("three stalls across two spans did not ask for minimal mode")
	}
	if o.Observe(now.Add(16*time.Millisecond), 90*time.Millisecond, true) {
		t.Fatal("the observer asked a second time")
	}
	if !o.Done() {
		t.Fatal("a decided observer still reports work")
	}
}

func TestFastStartNeverAsks(t *testing.T) {
	o := New()
	_, hits := run(o, time.Now(), ReadyWindow+ObserveWindow+time.Second, 4*time.Millisecond, true)
	if len(hits) != 0 {
		t.Fatalf("a fast start asked for a suggestion: %v", hits)
	}
	if !o.Done() {
		t.Fatal("the observation window did not end")
	}
}

func TestOneBurstDoesNotAsk(t *testing.T) {
	o := New()
	now, _ := run(o, time.Now(), ReadyWindow+time.Second, 4*time.Millisecond, true)
	// A single burst of stalls inside one span, then a quiet rest of the window.
	for i := 0; i < 6; i++ {
		now = now.Add(16 * time.Millisecond)
		o.Observe(now, 200*time.Millisecond, true)
	}
	_, hits := run(o, now, ObserveWindow, 4*time.Millisecond, true)
	if len(hits) != 0 {
		t.Fatalf("one burst was enough on its own: %v", hits)
	}
}

func TestBackgroundFramesAreNotObserved(t *testing.T) {
	o := New()
	now := time.Now()
	for i := 0; i < 5000; i++ {
		now = now.Add(16 * time.Millisecond)
		if o.Observe(now, 400*time.Millisecond, false) {
			t.Fatal("a background window asked for a suggestion")
		}
	}
	o.mu.Lock()
	ready, watched, stalls := o.ready, o.watched, o.stalls
	o.mu.Unlock()
	if ready != 0 || watched != 0 || stalls != 0 {
		t.Fatalf("background frames were counted: ready=%v watched=%v stalls=%d", ready, watched, stalls)
	}
	if o.Done() {
		t.Fatal("background frames ended the observation")
	}
}

func TestIdleGapIsNotStallEvidence(t *testing.T) {
	o := New()
	now, _ := run(o, time.Now(), ReadyWindow+time.Second, 4*time.Millisecond, true)
	// The machine slept or the window sat idle for ten minutes; the first frame
	// after it must not be read as ten minutes of observed time or a stall.
	now = now.Add(10 * time.Minute)
	if o.Observe(now, 4*time.Millisecond, true) {
		t.Fatal("an idle gap was treated as a suggestion")
	}
	o.mu.Lock()
	watched, stalls := o.watched, o.stalls
	o.mu.Unlock()
	if watched > ReadyWindow+time.Second+time.Second+MaxGap || stalls != 0 {
		t.Fatalf("the idle gap was credited: watched=%v stalls=%d", watched, stalls)
	}
}

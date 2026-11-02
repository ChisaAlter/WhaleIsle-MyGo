package main

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestBackgroundProbeServesLastAnswerAndRefreshesOffThread(t *testing.T) {
	var p backgroundProbe[int]
	var calls atomic.Int32
	answer := atomic.Int32{}
	answer.Store(1)
	probe := func() int { calls.Add(1); return int(answer.Load()) }
	changed := make(chan struct{}, 4)
	notify := func() { changed <- struct{}{} }

	if got := p.get("k", time.Hour, probe, notify); got != 1 || calls.Load() != 1 {
		t.Fatalf("first answer %d after %d probes", got, calls.Load())
	}
	if got := p.get("k", time.Hour, probe, notify); got != 1 || calls.Load() != 1 {
		t.Fatal("a fresh answer was probed again")
	}

	answer.Store(2)
	if got := p.get("k", 0, probe, notify); got != 1 {
		t.Fatalf("a stale read waited for the probe: %d", got)
	}
	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("a changed answer did not notify")
	}
	if got := p.get("k", time.Hour, probe, notify); got != 2 {
		t.Fatalf("the refreshed answer was not kept: %d", got)
	}

	before := calls.Load()
	if got := p.get("other", time.Hour, probe, notify); got != 2 || calls.Load() != before+1 {
		t.Fatal("a new key was not probed synchronously")
	}
	p.put("other", 7)
	if got := p.get("other", time.Hour, probe, notify); got != 7 {
		t.Fatalf("put answer not served: %d", got)
	}
	select {
	case <-changed:
		t.Fatal("an unchanged or synchronous answer notified")
	default:
	}
}

// slowRefresh starts a stale refresh of key whose probe blocks until release is
// closed and then reports answer, and returns once the probe is running.
func slowRefresh(t *testing.T, p *backgroundProbe[int], key string, answer int, release <-chan struct{}, notify func()) {
	t.Helper()
	started := make(chan struct{})
	p.get(key, 0, func() int {
		close(started)
		<-release
		return answer
	}, notify)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh did not start")
	}
}

func waitProbeIdle(t *testing.T, p *backgroundProbe[int]) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		p.mu.Lock()
		idle := !p.probing
		p.mu.Unlock()
		if idle {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the refresh did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestBackgroundProbeRefreshDoesNotOverwriteANewerPut(t *testing.T) {
	var p backgroundProbe[int]
	changed := make(chan struct{}, 4)
	notify := func() { changed <- struct{}{} }
	p.put("k", 1)

	release := make(chan struct{})
	slowRefresh(t, &p, "k", 2, release, notify) // began before the fresh read below
	p.put("k", 9)                               // an action read the file system after the refresh began
	close(release)
	waitProbeIdle(t, &p)

	if got := p.get("k", time.Hour, func() int { t.Fatal("served answer was probed again"); return 0 }, notify); got != 9 {
		t.Fatalf("an older refresh replaced the fresher put: %d", got)
	}
	select {
	case <-changed:
		t.Fatal("a discarded refresh notified")
	default:
	}
}

func TestBackgroundProbeRefreshDoesNotSurviveAKeyRoundTrip(t *testing.T) {
	var p backgroundProbe[int]
	changed := make(chan struct{}, 4)
	notify := func() { changed <- struct{}{} }
	p.put("a", 1)

	release := make(chan struct{})
	slowRefresh(t, &p, "a", 2, release, notify)
	// Away to another key and back: each first question is answered afresh, so the
	// refresh that began under the first "a" is out of date.
	if got := p.get("b", time.Hour, func() int { return 5 }, notify); got != 5 {
		t.Fatalf("other key: %d", got)
	}
	if got := p.get("a", time.Hour, func() int { return 3 }, notify); got != 3 {
		t.Fatalf("returning to the key did not probe it again: %d", got)
	}
	close(release)
	waitProbeIdle(t, &p)

	if got := p.get("a", time.Hour, func() int { t.Fatal("served answer was probed again"); return 0 }, notify); got != 3 {
		t.Fatalf("an out-of-date refresh replaced the answer: %d", got)
	}
	select {
	case <-changed:
		t.Fatal("a discarded refresh notified")
	default:
	}
}

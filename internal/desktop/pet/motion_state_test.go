package pet

import (
	"math"
	"testing"
	"time"
)

func TestWanderTargetMatchesOriginalUpperBandAndBounce(t *testing.T) {
	bounds := motionRect{x: -1000, y: 40, width: 1000, height: 1000}
	rolls := []float64{0.5, 0, 0.5}
	index := 0
	target := wanderTarget(motionPoint{x: -230, y: 500}, 220, 438, bounds, 1, 1, func() float64 {
		value := rolls[index]
		index++
		return value
	})
	// The 150-DIP rightward reach hits the edge and bounces left.
	if target.x != -380 || target.y != 161 {
		t.Fatalf("target = %+v, want {-380 161}", target)
	}
}

func TestWanderTargetSmallDisplayStaysReachable(t *testing.T) {
	bounds := motionRect{x: -90, y: -50, width: 160, height: 180}
	target := wanderTarget(motionPoint{x: 999, y: 999}, 220, 438, bounds, 1, 1, func() float64 { return 1 })
	if target != (motionPoint{x: -90, y: -50}) {
		t.Fatalf("oversized companion escaped work area: %+v", target)
	}
}

func TestWanderActivityContracts(t *testing.T) {
	for _, tc := range []struct {
		activity           string
		minWait, maxWait   time.Duration
		minSpeed, maxSpeed float64
	}{
		{"quiet", 600 * time.Second, 1200 * time.Second, 40, 60},
		{"balanced", 300 * time.Second, 540 * time.Second, 55, 75},
		{"active", 120 * time.Second, 240 * time.Second, 70, 90},
		{"unknown", 300 * time.Second, 540 * time.Second, 55, 75},
	} {
		if got := wanderInterval(tc.activity, 0); got != tc.minWait {
			t.Errorf("%s minimum interval = %v", tc.activity, got)
		}
		if got := wanderInterval(tc.activity, 1); got != tc.maxWait {
			t.Errorf("%s maximum interval = %v", tc.activity, got)
		}
		if got := wanderSpeed(tc.activity, 1, 0); got != tc.minSpeed {
			t.Errorf("%s minimum speed = %v", tc.activity, got)
		}
		if got := wanderSpeed(tc.activity, 1.6, 1); math.Abs(got-tc.maxSpeed/1.36) > 1e-9 {
			t.Errorf("%s scaled maximum speed = %v", tc.activity, got)
		}
	}
}

func TestWanderGlideUsesEaseAndExactEndpoint(t *testing.T) {
	g := newWanderGlide(motionPoint{x: 10, y: 20}, motionPoint{x: 110, y: 20}, 50)
	if g.duration != 2 {
		t.Fatalf("duration = %v", g.duration)
	}
	p, done := g.position(0.5)
	if p != (motionPoint{x: 22.5, y: 20}) || done {
		t.Fatalf("quarter glide = %+v, %v", p, done)
	}
	p, done = g.position(20)
	if p != (motionPoint{x: 110, y: 20}) || !done {
		t.Fatalf("endpoint = %+v, %v", p, done)
	}
	if g := newWanderGlide(motionPoint{}, motionPoint{}, 0); g.duration != 0.4 {
		t.Fatalf("minimum duration = %v", g.duration)
	}
}

func TestMotionFiniteClamps(t *testing.T) {
	for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := motionScale(invalid); got != 1 {
			t.Fatalf("invalid scale = %v", got)
		}
		target := wanderTarget(motionPoint{}, 220, 438, motionRect{width: 1920, height: 1080}, 1, invalid, func() float64 { return invalid })
		if math.IsNaN(target.x) || math.IsNaN(target.y) || math.IsInf(target.x, 0) || math.IsInf(target.y, 0) {
			t.Fatalf("nonfinite target = %+v", target)
		}
	}
	if motionScale(-1) != 0.6 || motionScale(1000) != 1.6 {
		t.Fatal("scale was not bounded")
	}
}

func TestMotionProvenanceNeverSavesProgrammaticEvents(t *testing.T) {
	var moves motionProvenance
	moves.expect(motionPoint{x: 100, y: 200})
	if moves.manual(motionPoint{x: 100, y: 200}, true) {
		t.Fatal("expected automatic target classified as manual")
	}
	// A backend may synchronously clamp or adjust SetPosition.
	moves.writing = true
	if moves.manual(motionPoint{x: 99, y: 199}, true) {
		t.Fatal("reentrant automatic move classified as manual")
	}
	moves.writing = false
	moves.expect(motionPoint{x: 99, y: 199})
	moves.expect(motionPoint{x: 120, y: 220})
	if moves.manual(motionPoint{x: 99, y: 199}, true) {
		t.Fatal("delayed automatic target classified as manual")
	}
	if moves.manual(motionPoint{x: 400, y: 300}, false) {
		t.Fatal("unattributed OS move classified as manual")
	}
	if !moves.manual(motionPoint{x: 400, y: 300}, true) {
		t.Fatal("user movement did not produce a manual candidate")
	}
}

func TestWanderPausePowerSaveBoundary(t *testing.T) {
	now := time.Unix(1000, 0)
	if wanderPaused(true, false, false, true, now.Add(-5*time.Minute), now) {
		t.Fatal("power save starts strictly after five minutes")
	}
	if !wanderPaused(true, false, false, true, now.Add(-5*time.Minute-time.Nanosecond), now) {
		t.Fatal("idle power save was ignored")
	}
	for _, flags := range [][3]bool{{false, false, false}, {true, true, false}, {true, false, true}} {
		if !wanderPaused(flags[0], flags[1], flags[2], false, now, now) {
			t.Fatal("disabled/sleeping/busy wander should be paused")
		}
	}
}

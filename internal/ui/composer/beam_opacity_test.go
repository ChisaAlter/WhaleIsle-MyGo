package composer

import (
	"testing"
	"time"
)

func effectOpacityNear(t *testing.T, got, want beamOpacities) {
	t.Helper()
	effectNear(t, got.stroke, want.stroke)
	effectNear(t, got.inner, want.inner)
	effectNear(t, got.bloom, want.bloom)
}

func TestBeamOpacityIndependentRetargets(t *testing.T) {
	origin := time.Unix(100, 0)
	style := readBeamStyle(nil)
	var timeline beamTimeline
	timeline.frame(origin, style, true, false)
	timeline.frame(origin.Add(time.Second), style, true, false)
	style.Bloom = 0
	frame := timeline.frame(origin.Add(time.Second), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, .36})
	if timeline.stroke.changed != origin || timeline.inner.changed != origin {
		t.Fatal("bloom change restarted other layers")
	}
	frame = timeline.frame(origin.Add(1210*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, .36 * (1 - .802403387584857)})
	frame = timeline.frame(origin.Add(1420*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, 0})
	style.Intensity = 50
	frame = timeline.frame(origin.Add(2*time.Second), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, 0})
	frame = timeline.frame(origin.Add(2210*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6 - .3*.802403387584857, .42 - .21*.802403387584857, 0})
	if timeline.bloom.changed != origin.Add(time.Second) {
		t.Fatal("unchanged zero bloom target restarted its transition")
	}
	frame = timeline.frame(origin.Add(2420*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.3, .21, 0})
}

func TestBeamOpacityInterruptedTargetsAndReversals(t *testing.T) {
	origin := time.Unix(100, 0)
	var layer beamOpacityTransition
	layer.retarget(origin, .6)
	before := layer.value(origin.Add(210 * time.Millisecond))
	got, _ := layer.retarget(origin.Add(210*time.Millisecond), .3)
	effectNear(t, got, before)
	if layer.duration != 420*time.Millisecond || layer.reversing != 1 {
		t.Fatal("unrelated target incorrectly shortened", layer)
	}
	effectNear(t, layer.value(origin.Add(420*time.Millisecond)), before+(.3-before)*.802403387584857)
	// Reverse toward this transition's adjusted start; shortening applies.
	got, _ = layer.retarget(origin.Add(420*time.Millisecond), before)
	effectNear(t, got, before+(.3-before)*.802403387584857)
	if layer.duration >= 420*time.Millisecond || layer.reverseStart != .3 {
		t.Fatal("reverse did not retain CSS adjusted start", layer)
	}
	// A repeated frame with the same target must not reset the start time.
	changed := layer.changed
	layer.retarget(origin.Add(430*time.Millisecond), before)
	if layer.changed != changed {
		t.Fatal("unchanged target restarted transition")
	}
	got, running := layer.retarget(changed.Add(layer.duration), before)
	effectNear(t, got, before)
	if running {
		t.Fatal("completed reverse is still running")
	}
}

func TestBeamOpacityBreathingRetargetAndDisable(t *testing.T) {
	origin := time.Unix(100, 0)
	style := readBeamStyle(nil)
	style.Breathing.Enabled, style.Breathing.Period, style.Breathing.Amplitude, style.Easing = true, 6, .3, "linear"
	var timeline beamTimeline
	timeline.frame(origin, style, true, false)
	timeline.frame(origin.Add(420*time.Millisecond), style, true, false)
	// Each new animated breathe-scale changes the element's opacity target.
	before := beamOpacities{timeline.stroke.value(origin.Add(630 * time.Millisecond)), timeline.inner.value(origin.Add(630 * time.Millisecond)), timeline.bloom.value(origin.Add(630 * time.Millisecond))}
	frame := timeline.frame(origin.Add(630*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, before)
	effectNear(t, timeline.stroke.target, .6*1.063)
	effectNear(t, timeline.inner.target, .42*1.063)
	effectNear(t, timeline.bloom.target, .36*1.063)
	if timeline.stroke.duration != beamOpacityDuration {
		t.Fatal("breathing retarget lost full duration")
	}
	before = beamOpacities{timeline.stroke.value(origin.Add(840 * time.Millisecond)), timeline.inner.value(origin.Add(840 * time.Millisecond)), timeline.bloom.value(origin.Add(840 * time.Millisecond))}
	style.Breathing.Enabled = false
	frame = timeline.frame(origin.Add(840*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, before)
	effectNear(t, timeline.stroke.target, .6)
	effectNear(t, timeline.inner.target, .42)
	effectNear(t, timeline.bloom.target, .36)
	if !timeline.breatheStarted.IsZero() {
		t.Fatal("disabled breathing retained its animation")
	}
	before = beamOpacities{timeline.stroke.value(origin.Add(1050 * time.Millisecond)), timeline.inner.value(origin.Add(1050 * time.Millisecond)), timeline.bloom.value(origin.Add(1050 * time.Millisecond))}
	frame = timeline.frame(origin.Add(1050*time.Millisecond), style, false, false)
	effectOpacityNear(t, frame.opacity, before)
	if timeline.stroke.target != 0 || timeline.inner.target != 0 || timeline.bloom.target != 0 {
		t.Fatal("inactive child retained dynamic opacity target")
	}
	frame = timeline.frame(origin.Add(1470*time.Millisecond), style, false, false)
	if frame.opacity != (beamOpacities{}) || frame.animate {
		t.Fatal("disabled dynamic layers did not finish fading", frame)
	}
}

func TestBeamOpacityNightSamplingAndRetarget(t *testing.T) {
	// Match the machine's local clock without mutating its timezone.
	origin := time.Date(2026, 10, 19, 21, 59, 40, 0, time.Local)
	style := readBeamStyle(nil)
	style.Night.Enabled = true
	var timeline beamTimeline
	timeline.frame(origin, style, true, false)
	timeline.frame(origin.Add(time.Second), style, true, false)
	frame := timeline.frame(origin.Add(21*time.Second), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, .36})
	if timeline.stroke.target != .6 {
		t.Fatal("unrelated repaint sampled local minute ahead of the source timer")
	}
	frame = timeline.frame(origin.Add(time.Minute), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.6, .42, .36})
	effectNear(t, timeline.stroke.target, .36)
	effectNear(t, timeline.inner.target, .252)
	effectNear(t, timeline.bloom.target, .216)
	frame = timeline.frame(origin.Add(time.Minute+420*time.Millisecond), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.36, .252, .216})
	style.Night.Dim = 80
	frame = timeline.frame(origin.Add(time.Minute+time.Second), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.36, .252, .216})
	effectNear(t, timeline.stroke.target, .12)
	style.Night.Enabled = false
	frame = timeline.frame(origin.Add(2*time.Minute), style, true, false)
	effectOpacityNear(t, frame.opacity, beamOpacities{.12, .084, .072})
	effectNear(t, timeline.stroke.target, .6)
}

func TestBeamOpacitySourceRoundingAndClamping(t *testing.T) {
	style := readBeamStyle(nil)
	style.Intensity, style.Bloom = 33.333, 77.777
	effectOpacityNear(t, beamOpacityTargets(style, true, 1, 1), beamOpacities{.2, .14, .0933})
	style.Intensity, style.Bloom = 200, 200
	effectOpacityNear(t, beamOpacityTargets(style, true, 1, 2), beamOpacities{1, 1, 1})
	effectOpacityNear(t, beamOpacityTargets(style, false, 1, 2), beamOpacities{})
}

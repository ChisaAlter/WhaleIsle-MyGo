package composer

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// ui-theme/src/styles/base.css supplies these tokens to TypingFxLayer.
const typingFlipDuration = 400 * time.Millisecond
const typingCaretPeriod = typingFlipDuration * 5 / 2
const beamOpacityDuration = 420 * time.Millisecond

// cssBezier solves the x coordinate before sampling y. Evaluating y directly
// at progress, as polynomial ease helpers do, is not CSS cubic-bezier timing.
func cssBezier(progress, x1, y1, x2, y2 float64) float64 {
	if progress <= 0 {
		return 0
	}
	if progress >= 1 {
		return 1
	}
	coordinate := func(t, a, b float64) float64 {
		u := 1 - t
		return 3*u*u*t*a + 3*u*t*t*b + t*t*t
	}
	lo, hi, t := 0.0, 1.0, progress
	for range 24 {
		x := coordinate(t, x1, x2)
		if math.Abs(x-progress) < 1e-8 {
			break
		}
		if x < progress {
			lo = t
		} else {
			hi = t
		}
		t = (lo + hi) / 2
	}
	return coordinate(t, y1, y2)
}

func typingEase(progress float64) float64 { return cssBezier(progress, .4, 0, .2, 1) }

func beamEase(progress float64, easing string) float64 {
	switch easing {
	case "ease":
		return cssBezier(progress, .25, .1, .25, 1)
	case "ease-in-out":
		return cssBezier(progress, .42, 0, .58, 1)
	case "cubic-bezier":
		return typingEase(progress)
	default:
		return min(1, max(0, progress))
	}
}

func typingDuration(speed float64) time.Duration {
	return time.Duration(float64(typingFlipDuration) * 100 / speed)
}

// CSS builds a separate keyframe sequence for each property: the drop's 35%
// opacity stop does not interrupt its 0..55% transform interpolation.
func typingKeyframe(effect string, progress float64) (alpha, offsetEm, scale float32) {
	p := min(1, max(0, progress))
	eased := typingEase(p)
	scale = 1
	switch effect {
	case "rise":
		alpha, offsetEm = float32(.9*(1-eased)), float32(-.7*eased)
	case "flash":
		if p <= .2 {
			alpha = float32(typingEase(p / .2))
		} else {
			alpha = float32(1 - typingEase((p-.2)/.8))
		}
		scale = float32(1 + .35*(1-eased))
	default:
		if p <= .35 {
			alpha = float32(typingEase(p / .35))
		} else {
			alpha = float32(1 - typingEase((p-.35)/.65))
		}
		travel := 1 - typingEase(p/.55)
		offsetEm, scale = float32(-.55*travel), float32(1+.18*travel)
	}
	return
}

// step-end holds opacity 1 through the 45..50% segment, then 0 until
// the cycle boundary. Schedule that boundary, rather than a drifting interval.
func typingCaretBlink(elapsed time.Duration) (visible bool, next time.Duration) {
	phase := elapsed % typingCaretPeriod
	if phase < typingCaretPeriod/2 {
		return true, typingCaretPeriod/2 - phase
	}
	return false, typingCaretPeriod - phase
}

type beamTimeline struct {
	palette        []ui.Color
	active         bool
	started        time.Time
	hueStarted     time.Time
	breatheStarted time.Time
	stroke         beamOpacityTransition
	inner          beamOpacityTransition
	bloom          beamOpacityTransition
	nightAt        time.Time
	nightNext      time.Time
	nightSampling  bool
}

type beamOpacities struct{ stroke, inner, bloom float64 }

type beamFrame struct {
	opacity             beamOpacities
	seconds, hueSeconds float64
	animate             bool
}

// Each CSS element owns a transition. A bloom-only change must not restart
// stroke/inner, and a new target starts at the current interpolated opacity.
type beamOpacityTransition struct {
	changed                    time.Time
	from, target, reverseStart float64
	duration                   time.Duration
	reversing                  float64
}

func (s *beamOpacityTransition) value(now time.Time) float64 {
	if s.duration == 0 {
		return s.target
	}
	progress := float64(now.Sub(s.changed)) / float64(s.duration)
	return s.from + (s.target-s.from)*beamEase(progress, "ease")
}

func (s *beamOpacityTransition) running(now time.Time) bool {
	return s.duration > 0 && now.Sub(s.changed) < s.duration
}

func (s *beamOpacityTransition) retarget(now time.Time, target float64) (value float64, running bool) {
	value = s.value(now)
	if target != s.target {
		if value == target {
			*s = beamOpacityTransition{target: target}
			return value, false
		}
		duration, reversing, reverseStart := beamOpacityDuration, 1.0, value
		if s.running(now) && target == s.reverseStart {
			progress := float64(now.Sub(s.changed)) / float64(s.duration)
			reversing = min(1, math.Abs(beamEase(progress, "ease")*s.reversing+1-s.reversing))
			duration = time.Duration(float64(beamOpacityDuration) * reversing)
			reverseStart = s.target
		}
		*s = beamOpacityTransition{changed: now, from: value, target: target, reverseStart: reverseStart, duration: duration, reversing: reversing}
	}
	return s.value(now), s.running(now)
}

func beamNightFactor(style beamStyle, now time.Time) float64 {
	if !style.Night.Enabled {
		return 1
	}
	local := now.Local()
	minute := local.Hour()*60 + local.Minute()
	from, to := style.Night.FromMinute, style.Night.ToMinute
	inside := from == to || from < to && minute >= from && minute < to || from > to && (minute >= from || minute < to)
	if inside {
		return 1 - style.Night.Dim/100
	}
	return 1
}

func beamOpacityTargets(style beamStyle, active bool, night, breathing float64) beamOpacities {
	if !active {
		return beamOpacities{}
	}
	// ComposerBeam.tsx rounds the CSS variables before CSS multiplies them.
	round := func(value float64) float64 { return math.Round(value*10000) / 10000 }
	intensity := style.Intensity / 100
	factor := round(night) * breathing
	opacity := func(base float64) float64 { return min(1, max(0, round(base)*factor)) }
	return beamOpacities{stroke: opacity(.6 * intensity), inner: opacity(.42 * intensity), bloom: opacity(.36 * intensity * style.Bloom / 100)}
}

// The source samples Date.now at mount, when active/night become enabled,
// and on their 60s interval. An unrelated frame must not advance that clock.
func (s *beamTimeline) sampleNight(now time.Time, style beamStyle) float64 {
	sampling := s.active && style.Night.Enabled
	if s.nightAt.IsZero() || sampling && !s.nightSampling {
		s.nightAt, s.nightNext = now, now.Add(time.Minute)
	} else if sampling && !now.Before(s.nightNext) {
		s.nightAt = now
		s.nightNext = s.nightNext.Add((now.Sub(s.nightNext)/time.Minute + 1) * time.Minute)
	}
	s.nightSampling = sampling
	return beamNightFactor(style, s.nightAt)
}

// Each named child animation has its own activation time. Toggling hue or
// breathing while the parent spins must not inherit the parent's old phase.
func (s *beamTimeline) effectTimes(now time.Time, style beamStyle) (breathe, hue float64) {
	if !s.active {
		s.hueStarted, s.breatheStarted = time.Time{}, time.Time{}
		return
	}
	if !style.HueCycle.Enabled {
		s.hueStarted = time.Time{}
	} else {
		if s.hueStarted.IsZero() {
			s.hueStarted = now
		}
		hue = now.Sub(s.hueStarted).Seconds()
	}
	if !style.Breathing.Enabled {
		s.breatheStarted = time.Time{}
	} else {
		if s.breatheStarted.IsZero() {
			s.breatheStarted = now
		}
		breathe = now.Sub(s.breatheStarted).Seconds()
	}
	return
}

func (s *beamTimeline) frame(now time.Time, style beamStyle, active, reduced bool) (frame beamFrame) {
	if reduced {
		*s = beamTimeline{}
		return
	}
	if active != s.active {
		s.active = active
		if active {
			s.started = now
		}
	}
	breatheSeconds, hueSeconds := s.effectTimes(now, style)
	breathing := 1.0
	if active {
		frame.seconds = now.Sub(s.started).Seconds()
		breathing = beamBreathing(breatheSeconds, style)
	}
	frame.hueSeconds = hueSeconds
	target := beamOpacityTargets(style, active, s.sampleNight(now, style), breathing)
	var stroke, inner, bloom bool
	frame.opacity.stroke, stroke = s.stroke.retarget(now, target.stroke)
	frame.opacity.inner, inner = s.inner.retarget(now, target.inner)
	frame.opacity.bloom, bloom = s.bloom.retarget(now, target.bloom)
	frame.animate = active || stroke || inner || bloom
	return
}

func beamPhase(seconds float64, style beamStyle) float64 {
	iteration := seconds / style.Period
	phase := iteration - math.Floor(iteration)
	if style.Direction == "counterclockwise" || style.Direction == "pingPong" && int64(iteration)%2 != 0 {
		phase = 1 - phase
	}
	return beamEase(phase, style.Easing)
}

func beamBreathing(seconds float64, style beamStyle) float64 {
	if !style.Breathing.Enabled {
		return 1
	}
	iteration := seconds / style.Breathing.Period
	phase := iteration - math.Floor(iteration)
	if int64(iteration)%2 != 0 {
		phase = 1 - phase
	}
	if phase <= .5 {
		return 1 + style.Breathing.Amplitude*beamEase(phase*2, style.Easing)
	}
	return 1 + style.Breathing.Amplitude*(1-beamEase((phase-.5)*2, style.Easing))
}

func beamHue(seconds float64, style beamStyle) float64 {
	// Without the hue animation the source has no hue-rotate filter.
	if !style.HueCycle.Enabled {
		return 0
	}
	period := 12.0
	if style.Breathing.Enabled {
		period = style.HueCycle.Period
	}
	phase := math.Mod(seconds/period, 1)
	if phase > .5 {
		phase = 1 - phase
	}
	return style.Hue + style.HueCycle.Range*(2*beamEase(phase*2, "ease-in-out")-1)
}

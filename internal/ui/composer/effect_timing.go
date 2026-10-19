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
	changed        time.Time
	from, target   float64
	duration       time.Duration
	reversing      float64
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

func (s *beamTimeline) opacity(now time.Time) float64 {
	if s.duration == 0 {
		return s.target
	}
	p := float64(now.Sub(s.changed)) / float64(s.duration)
	return s.from + (s.target-s.from)*beamEase(p, "ease")
}

func (s *beamTimeline) update(now time.Time, active, reduced bool) (opacity, elapsed float64, animate bool) {
	if reduced {
		*s = beamTimeline{}
		return
	}
	if active != s.active {
		current := s.opacity(now)
		reversing := 1.0
		if s.duration > 0 && now.Sub(s.changed) < s.duration {
			p := float64(now.Sub(s.changed)) / float64(s.duration)
			// CSS reversing-adjusted start value shortens interrupted reversals.
			reversing = math.Abs(beamEase(p, "ease")*s.reversing + 1 - s.reversing)
		}
		s.active, s.from, s.changed, s.reversing = active, current, now, reversing
		s.duration = time.Duration(float64(beamOpacityDuration) * reversing)
		s.target = 0
		if active {
			s.target, s.started = 1, now
		}
	}
	opacity = s.opacity(now)
	if active {
		elapsed = now.Sub(s.started).Seconds()
	}
	animate = active || now.Sub(s.changed) < s.duration
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

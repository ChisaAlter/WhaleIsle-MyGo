package pet

import (
	"math"
	"time"
)

const motionCoordinateLimit = 1 << 20

type motionPoint struct{ x, y float64 }
type motionRect struct{ x, y, width, height float64 }

func motionFinite(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}
func motionClamp(value, lo, hi float64) float64 {
	return math.Min(math.Max(motionFinite(value, lo), lo), math.Max(lo, hi))
}
func motionScale(value float64) float64 {
	return motionClamp(motionFinite(value, 1), 0.6, 1.6)
}
func motionRoll(value float64) float64 { return motionClamp(value, 0, 1) }

func boundedMotionRect(bounds motionRect) motionRect {
	bounds.x = motionClamp(bounds.x, -motionCoordinateLimit, motionCoordinateLimit)
	bounds.y = motionClamp(bounds.y, -motionCoordinateLimit, motionCoordinateLimit)
	bounds.width = motionClamp(bounds.width, 1, motionCoordinateLimit)
	bounds.height = motionClamp(bounds.height, 1, motionCoordinateLimit)
	return bounds
}
func clampMotionPoint(point motionPoint, width, height float64, bounds motionRect) motionPoint {
	bounds = boundedMotionRect(bounds)
	width = motionClamp(width, 1, motionCoordinateLimit)
	height = motionClamp(height, 1, motionCoordinateLimit)
	return motionPoint{
		x: motionClamp(point.x, bounds.x, bounds.x+bounds.width-width),
		y: motionClamp(point.y, bounds.y, bounds.y+bounds.height-height),
	}
}

// The native window is the character's movable footprint. These formulas
// port pet-wander.js from WhaleIsle 90465b6; there is no renderer coordinate API.
func wanderTarget(from motionPoint, width, height float64, bounds motionRect, facing int, scale float64, random func() float64) motionPoint {
	if random == nil {
		random = func() float64 { return 0.5 }
	}
	bounds = boundedMotionRect(bounds)
	from = clampMotionPoint(from, width, height, bounds)
	width = motionClamp(width, 1, motionCoordinateLimit)
	height = motionClamp(height, 1, motionCoordinateLimit)
	reach := (60 + motionRoll(random())*180) * motionScale(scale)
	direction := 1.0
	if facing < 0 {
		direction = -1
	}
	if motionRoll(random()) >= 0.7 {
		direction = -direction
	}
	target := motionPoint{x: from.x + direction*reach, y: from.y}
	hi := bounds.x + bounds.width - width
	if target.x < bounds.x || target.x > hi {
		target.x = from.x - direction*reach
	}
	target.x = motionClamp(target.x, bounds.x, hi)
	top := bounds.y + bounds.height*0.08
	bottom := bounds.y + bounds.height*0.6 - height
	if bottom > top {
		target.y = top + motionRoll(random())*(bottom-top)
	}
	return clampMotionPoint(target, width, height, bounds)
}

func wanderInterval(activity string, roll float64) time.Duration {
	lo, hi := 300.0, 540.0
	switch activity {
	case "quiet":
		lo, hi = 600, 1200
	case "active":
		lo, hi = 120, 240
	}
	return time.Duration((lo + motionRoll(roll)*(hi-lo)) * float64(time.Second))
}
func wanderSpeed(activity string, scale, roll float64) float64 {
	base := 55.0
	switch activity {
	case "quiet":
		base = 40
	case "active":
		base = 70
	}
	return (base + motionRoll(roll)*20) / math.Max(1, motionScale(scale)*0.85)
}
func wanderPaused(enabled, sleeping, busy, powerSave bool, lastContact, now time.Time) bool {
	return !enabled || sleeping || busy || (powerSave && now.Sub(lastContact) > 5*time.Minute)
}

type wanderGlide struct {
	from, to motionPoint
	duration float64
}

func newWanderGlide(from, to motionPoint, speed float64) wanderGlide {
	duration := math.Hypot(to.x-from.x, to.y-from.y) / math.Max(20, motionFinite(speed, 60))
	return wanderGlide{from: from, to: to, duration: motionClamp(duration, 0.4, 120)}
}
func (g wanderGlide) position(elapsed float64) (motionPoint, bool) {
	t := motionClamp(elapsed/math.Max(0.001, g.duration), 0, 1)
	ease := 2 * t * t
	if t >= 0.5 {
		ease = 1 - math.Pow(-2*t+2, 2)/2
	}
	return motionPoint{x: g.from.x + (g.to.x-g.from.x)*ease, y: g.from.y + (g.to.y-g.from.y)*ease}, t >= 1
}

// Only a position attributed to recent native/page interaction can be saved.
// SetPosition and its synchronous/clamped/delayed notifications never become
// anchors. OnMove reads current Bounds, so a coalesced event still matches
// the most recent expected point; the bounded history also rejects older ones.
type motionProvenance struct {
	expected    [64]motionPoint
	count, next int
	writing     bool
}

func (m *motionProvenance) expect(point motionPoint) {
	m.expected[m.next] = point
	m.next = (m.next + 1) % len(m.expected)
	m.count = min(m.count+1, len(m.expected))
}
func (m *motionProvenance) manual(point motionPoint, contact bool) bool {
	if m.writing || !contact {
		return false
	}
	for index := 0; index < m.count; index++ {
		expected := m.expected[index]
		if math.Abs(point.x-expected.x) < 0.5 && math.Abs(point.y-expected.y) < 0.5 {
			return false
		}
	}
	return true
}

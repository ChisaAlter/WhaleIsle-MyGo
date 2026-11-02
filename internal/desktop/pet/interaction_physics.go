package pet

import (
	"math"
	"time"
)

// Physics lives in native desktop space; reference constants, release trails
// and renderer velocity are in CSS DIPs. Scale distance/velocity/acceleration
// once, leaving seconds, restitution and friction unchanged.
func nativeInteractionRelease(trail []interactionSample, now time.Time, dpi float64) (float64, float64) {
	if dpi <= 0 {
		dpi = 1
	}
	if len(trail) == 0 {
		return 0, 0
	}
	origin := trail[0]
	logical := make([]interactionSample, len(trail))
	for i, sample := range trail {
		logical[i] = interactionSample{at: sample.at, x: (sample.x - origin.x) / dpi, y: (sample.y - origin.y) / dpi}
	}
	vx, vy := interactionRelease(logical, now)
	return vx * dpi, vy * dpi
}

func nativeThrowStep(position, velocity motionPoint, dt float64, area motionRect, width, height, dpi float64) (motionPoint, motionPoint, bool) {
	if dpi <= 0 {
		dpi = 1
	}
	left, top := area.x, area.y
	right, floor := math.Max(left, area.x+area.width-width), math.Max(top, area.y+area.height-height)
	velocity.y += 1400 * dpi * dt
	position.x += velocity.x * dt
	position.y += velocity.y * dt
	if position.x < left {
		position.x = left
		velocity.x = math.Abs(velocity.x) * .78
	}
	if position.x > right {
		position.x = right
		velocity.x = -math.Abs(velocity.x) * .78
	}
	if position.y < top {
		position.y = top
		velocity.y = math.Abs(velocity.y) * .78
	}
	if position.y >= floor {
		position.y = floor
		velocity.x *= math.Max(0, 1-2.5*dt)
		if math.Abs(velocity.y) < 40*dpi {
			velocity.y = 0
		} else {
			velocity.y = -math.Abs(velocity.y) * .78
		}
	}
	rested := position.y >= floor-dpi && math.Abs(velocity.y) < dpi && math.Abs(velocity.x) < 15*dpi
	return position, velocity, rested
}

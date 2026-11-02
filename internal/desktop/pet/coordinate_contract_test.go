package pet

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func closeCoordinate(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-8 {
		t.Fatalf("%s=%g want %g", name, got, want)
	}
}

func TestCoordinateMotionKeepsCSSUnitsAt100And150Percent(t *testing.T) {
	now := time.Unix(10000, 0)
	for _, dpi := range []float64{1, 1.5} {
		area := motionRect{x: -3000, y: -600, width: 1200 * dpi, height: 1000 * dpi}
		from := motionPoint{x: area.x + 300*dpi, y: area.y + 500*dpi}
		rolls := []float64{.5, 0, .5}
		idx := 0
		target := nativeWanderTarget(from, 240*dpi, 260*dpi, area, 1, 1, dpi, func() float64 { v := rolls[idx]; idx++; return v })
		closeCoordinate(t, "wander reach DIP", (target.x-from.x)/dpi, 150)
		closeCoordinate(t, "wander target y DIP", (target.y-area.y)/dpi, 210)
		glide := newWanderGlide(from, target, wanderSpeed("balanced", 1, .5)*dpi)
		closeCoordinate(t, "glide seconds", glide.duration, math.Hypot(150, 290)/65)
		position, velocity, _ := nativeThrowStep(motionPoint{x: area.x + 100*dpi, y: area.y + 100*dpi}, motionPoint{x: 300 * dpi, y: -200 * dpi}, .05, area, 240*dpi, 260*dpi, dpi)
		closeCoordinate(t, "throw x DIP", (position.x-area.x)/dpi, 115)
		closeCoordinate(t, "throw y DIP", (position.y-area.y)/dpi, 93.5)
		closeCoordinate(t, "throw vx DIP/s", velocity.x/dpi, 300)
		closeCoordinate(t, "throw vy DIP/s", velocity.y/dpi, -130)
		trail := []interactionSample{{at: now.Add(-100 * time.Millisecond), x: -400 * dpi, y: 100 * dpi}, {at: now.Add(-50 * time.Millisecond), x: -350 * dpi, y: 100 * dpi}, {at: now, x: -300 * dpi, y: 100 * dpi}}
		vx, vy := nativeInteractionRelease(trail, now, dpi)
		closeCoordinate(t, "release vx DIP/s", vx/dpi, 4800*(1-math.Exp(-1000.0/4800)))
		closeCoordinate(t, "release vy DIP/s", vy, 0)
		// Rest and dead-zone thresholds must scale with distance, not time.
		floor := area.y + area.height - 260*dpi
		_, _, rested := nativeThrowStep(motionPoint{x: area.x + 100*dpi, y: floor}, motionPoint{x: 10 * dpi, y: 0}, .01, area, 240*dpi, 260*dpi, dpi)
		if !rested {
			t.Fatalf("%g scale did not settle equivalent CSS velocity", dpi)
		}
		t.Logf("dpi=%g reach=%.3fpx/150DIP speed=%.3fpx/s/65DIP/s gravity=%.3fpx/s²/1400DIP/s² glide=%.9fs throw=(%.3f,%.3f)DIP release=%.9fDIP/s", dpi, 150*dpi, 65*dpi, 1400*dpi, glide.duration, (position.x-area.x)/dpi, (position.y-area.y)/dpi, vx/dpi)
	}
}

func TestCoordinateLayoutRoundTripHitAt100And150Percent(t *testing.T) {
	for _, dpi := range []float64{1, 1.5} {
		origin := mygo.Rectangle{X: -2500, Y: -300, Width: int(1200 * dpi), Height: int(900 * dpi)}
		stage := mygo.Rectangle{X: origin.X + int(100*dpi), Y: origin.Y + int(80*dpi), Width: int(240 * dpi), Height: int(260 * dpi)}
		area := origin
		layout := localPetLayout(stage, origin, area, dpi)
		if layout.X != 100 || layout.Y != 80 || layout.Width != 240 || layout.Height != 260 {
			t.Fatalf("%g layout=%+v", dpi, layout)
		}
		report := petHitReport{Width: 1200, Height: 900, Regions: []petHitRect{{X: 100, Y: 80, Width: 240, Height: 260}}}
		inside := mygo.Point{X: stage.X + int(60*dpi), Y: stage.Y + int(60*dpi)}
		outside := mygo.Point{X: stage.X - int(9*dpi) - 1, Y: stage.Y}
		if !petHitContainsScaled(report, origin, inside, false, dpi) || petHitContainsScaled(report, origin, outside, false, dpi) {
			t.Fatalf("%g physical/CSS hit disagreement", dpi)
		}
		closeCoordinate(t, "cursor local CSS", float64(inside.X-origin.X)/dpi, 160)
		t.Logf("dpi=%g actual-origin=(%d,%d) stage=%+v CSS=%+v cursorCSS=(160,140) hit-in=true hit-out=false", dpi, origin.X, origin.Y, stage, layout)
	}
}

package chat

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Chromium's desktop programmatic scroll curve. Duration grows with the square
// root of distance, at 60 steps/second, and is capped at 1.5 seconds.
func inspectorChatScrollDuration(distance float32) time.Duration {
	return time.Duration(min(math.Sqrt(math.Abs(float64(distance)))/60, 1.5) * float64(time.Second))
}

// Cubic-bezier(.4,0,0,1): invert x to find the corresponding y. The existing
// toolkit cubic EaseOut has a different acceleration from Chromium's curve.
func inspectorChatScrollEase(progress float32) float32 {
	if progress <= 0 {
		return 0
	}
	if progress >= 1 {
		return 1
	}
	x := float64(progress)
	lo, hi := float64(0), float64(1)
	for range 24 {
		t := (lo + hi) / 2
		u := 1 - t
		bx := 1.2*u*u*t + t*t*t
		if bx < x {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	return float32(3*(1-t)*t*t + t*t*t)
}

func inspectorChatRevealInput(c *ui.Context, n *InspectorNavigation) {
	if n == nil {
		return
	}
	services := c.Services()
	c.ObserveInput(func(ev ui.InputEvent) {
		r := &n.reveal
		if !r.pending || !(r.waiting || r.centerQueued) {
			return
		}
		interrupt := ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputScroll
		if ev.Kind == ui.InputKeyDown {
			switch ev.Key {
			case ui.KeyUp, ui.KeyDown, ui.KeyPageUp, ui.KeyPageDown, ui.KeyHome, ui.KeyEnd, ui.KeySpace, ui.KeyEscape:
				interrupt = true
			}
		}
		if interrupt {
			n.CancelReveal()
			services.Invalidate()
		}
	})
}

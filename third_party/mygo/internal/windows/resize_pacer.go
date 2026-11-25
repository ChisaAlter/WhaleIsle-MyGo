//go:build windows && (amd64 || arm64)

package windows

import "time"

const (
	// resizeThrottleMin and resizeThrottleMax bound the real-time resize redraw
	// interval so live window dragging tracks monitor VSync (16-33ms) instead of
	// freezing old frames or saturating the message queue on high-Hz mice.
	resizeThrottleMin = 16 * time.Millisecond
	resizeThrottleMax = 33 * time.Millisecond
)

// resizeIntervalForRefreshRate converts a display refresh rate in Hz into the
// live-resize redraw throttle interval, clamped to [16ms, 33ms].
func resizeIntervalForRefreshRate(hz float64) time.Duration {
	if hz <= 0 {
		return resizeThrottleMin
	}
	d := time.Duration(float64(time.Second) / hz)
	if d < resizeThrottleMin {
		return resizeThrottleMin
	}
	if d > resizeThrottleMax {
		return resizeThrottleMax
	}
	return d
}

// resizePacer throttles live WM_SIZE redraws to at most one frame per interval
// while guaranteeing an immediate first resize frame and a final flush when the
// timer fires or the interactive drag ends.
type resizePacer struct {
	lastPaint time.Time
	pending   bool
	timerOn   bool
}

// onSize records a non-empty WM_SIZE at now and reports whether the surface
// should redraw immediately or arm a one-shot timer for armDelay.
func (p *resizePacer) onSize(now time.Time, interval time.Duration) (drawNow bool, armDelay time.Duration) {
	if interval <= 0 {
		interval = resizeThrottleMin
	}
	if p.lastPaint.IsZero() || now.Sub(p.lastPaint) >= interval {
		p.lastPaint = now
		p.pending = false
		p.timerOn = false
		return true, 0
	}
	p.pending = true
	if p.timerOn {
		return false, 0
	}
	p.timerOn = true
	rem := interval - now.Sub(p.lastPaint)
	if rem < time.Millisecond {
		rem = time.Millisecond
	}
	return false, rem
}

// onTimer fires when the resize throttle timer expires and reports whether a
// deferred resize redraw is still needed.
func (p *resizePacer) onTimer(now time.Time) bool {
	p.timerOn = false
	if !p.pending {
		return false
	}
	p.pending = false
	p.lastPaint = now
	return true
}

// onExitSizeMove flushes any deferred resize at the end of a live drag so the
// released window always shows its exact final layout without delay.
func (p *resizePacer) onExitSizeMove(now time.Time) bool {
	p.timerOn = false
	if !p.pending {
		return false
	}
	p.pending = false
	p.lastPaint = now
	return true
}

// onPainted records that a surface paint just occurred for the current size,
// satisfying any deferred resize.
func (p *resizePacer) onPainted(now time.Time) (killTimer bool) {
	killTimer = p.timerOn
	p.lastPaint = now
	p.pending = false
	p.timerOn = false
	return killTimer
}

func (p *resizePacer) reset() {
	*p = resizePacer{}
}

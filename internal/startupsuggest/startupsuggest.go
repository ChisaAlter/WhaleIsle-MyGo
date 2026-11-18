// Package startupsuggest decides once per launch whether the interface started
// out slow enough to offer 极简模式 (minimal mode).
//
// It observes cost the application already measures — how long building one
// view frame took — so it needs no timer, goroutine or sampling loop of its own:
// a fast start pays a few comparisons per frame, and a finished observer pays
// one. Frames that arrive while the window is in the background are not
// observed, and a gap between frames is credited at most MaxGap, so an idle
// minute or a machine waking from sleep is never evidence of a slow interface.
//
// The shape follows the reference desktop's offer: the interface has to have
// been usable in the foreground for a while before observing starts, the
// observation covers a bounded window, and the offer is made only when stalls
// cluster across separate parts of that window. One slow frame is not a
// suggestion.
package startupsuggest

import (
	"sync"
	"time"
)

const (
	// ReadyWindow is how long the interface must have been usable in the
	// foreground before observing starts, so a still-appearing window is not
	// measured.
	ReadyWindow = 10 * time.Second
	// ObserveWindow bounds the observation, after which the answer is final.
	ObserveWindow = 60 * time.Second
	// StallThreshold is the view build cost past which a frame counts as a
	// stall: the same 16ms the interface's own slow-frame counter uses.
	StallThreshold = 16 * time.Millisecond
	// StallSpan is the length of one part of the observation window, and
	// MinStalls/MinSpans require that many stalls across at least that many
	// separate parts, so one burst of long frames is not a suggestion.
	StallSpan = 15 * time.Second
	MinStalls = 3
	MinSpans  = 2
	// MaxGap is the most observation time one frame is credited with. Frames
	// are the only clock the observer has; a longer gap was an idle window or a
	// sleeping machine, not a stutter it measured.
	MaxGap = 250 * time.Millisecond
)

// Observer accumulates startup responsiveness. Its zero value is not usable;
// make one with New.
type Observer struct {
	mu       sync.Mutex
	last     time.Time
	ready    time.Duration
	watched  time.Duration
	stalls   int
	spanHits []int
	decided  bool
	suggest  bool
}

// New returns an observer for one launch.
func New() *Observer { return &Observer{} }

// Observe reports one built frame: when it happened, how long building the view
// took, and whether the window was in the foreground. It returns true exactly
// once, on the frame that decides the interface started slow enough to offer
// minimal mode; it returns false for every other frame, including all of them
// once the question is settled.
func (o *Observer) Observe(now time.Time, build time.Duration, foreground bool) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.decided {
		return false
	}
	if !foreground {
		// The gap around a hidden or unfocused window is not observed time.
		o.last = time.Time{}
		return false
	}
	var gap time.Duration
	if !o.last.IsZero() {
		gap = now.Sub(o.last)
		switch {
		case gap < 0:
			gap = 0
		case gap > MaxGap:
			gap = MaxGap
		}
	}
	o.last = now
	if o.ready < ReadyWindow {
		o.ready += gap
		if o.ready < ReadyWindow {
			return false
		}
		// This frame closes the ready window; observation starts with the next.
		return false
	}
	o.watched += gap
	if build > StallThreshold {
		o.stalls++
		span := int(o.watched / StallSpan)
		seen := false
		for _, hit := range o.spanHits {
			if hit == span {
				seen = true
			}
		}
		if !seen {
			o.spanHits = append(o.spanHits, span)
		}
	}
	if o.stalls >= MinStalls && len(o.spanHits) >= MinSpans {
		o.decided, o.suggest = true, true
		return true
	}
	if o.watched >= ObserveWindow {
		o.decided = true
	}
	return false
}

// Done reports that the observation is over and Observe no longer needs
// calling.
func (o *Observer) Done() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.decided
}

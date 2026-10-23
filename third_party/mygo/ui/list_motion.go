package ui

import (
	"math"
	"time"
)

// ScrollAnimation describes a programmatic List scroll. Duration receives the
// measured distance in DIPs as layout resolves the target; it may be called
// more than once in the first frame and should have no side effects. A nil Duration or
// nonpositive result scrolls immediately. Ease maps elapsed time from 0 to 1;
// nil means Linear. Desktop reduced motion always scrolls immediately.
type ScrollAnimation struct {
	Duration func(distance float32) time.Duration
	Ease     Easing
}

type listScrollMotion struct {
	row            int
	key            any
	align          Align
	region         bool
	top, height    float64
	spec           ScrollAnimation
	started        time.Time
	initialized    bool
	initialFrame   uint64
	initialAnchor  int
	initialInset   float64
	completedFrame uint64
	from, last     float64
	duration       time.Duration
}

// ScrollToAnimated measures row and scrolls it to align over display frames.
// It tracks the target by Key when configured. Another scroll request replaces
// the animation. Removing the target or changing the offset externally stops it.
// Call once per navigation, rather than on every animation frame.
func (s *ListState) ScrollToAnimated(row int, align Align, animation ScrollAnimation) {
	s.req = listRequest{}
	s.motion = &listScrollMotion{row: row, align: align, spec: animation}
	if s.Key != nil && row >= 0 && row < s.heights.n {
		s.motion.key = s.Key(row)
	}
}

// ScrollToRegionAnimated measures row and scrolls a region within it to align.
// top and height are DIPs relative to the top of the row's content. Negative
// values become zero. At each layout, top is clamped to the measured row height,
// then height is clamped to the space remaining below top; a zero height targets
// a point. The list's content bounds can limit the requested alignment.
// It uses ScrollToAnimated's timing, Key tracking and cancellation behavior,
// including immediate scrolling for reduced motion. Call once per navigation.
func (s *ListState) ScrollToRegionAnimated(row int, top, height float32, align Align, animation ScrollAnimation) {
	s.ScrollToAnimated(row, align, animation)
	s.motion.region = true
	s.motion.top, s.motion.height = float64(max(0, top)), float64(max(0, height))
}

// StopScrollAnimation stops future movement at the last committed position.
// It adds no offset request, so an input event that follows can still scroll.
func (s *ListState) StopScrollAnimation() { s.motion = nil }

// ScrollAnimating reports whether a requested scroll is awaiting layout or moving.
func (s *ListState) ScrollAnimating() bool { return s.motion != nil && s.motion.completedFrame == 0 }

// ScrollOffset is the vertical offset committed by the last List layout in DIPs.
// It is independent of which ScrollState currently tracks the list. Input not
// laid out yet is reflected by the next layout.
func (s *ListState) ScrollOffset() float32 { return float32(s.wroteY) }

func (s *ListState) syncMotion(n int) {
	m := s.motion
	if m == nil {
		return
	}
	if m.key != nil && s.Key != nil {
		row, ok := s.find(m.key, m.row, n)
		if !ok {
			s.StopScrollAnimation()
			return
		}
		m.row = row
	} else if m.row < 0 || m.row >= n {
		s.StopScrollAnimation()
	} else if s.Key != nil {
		m.key = s.Key(m.row)
	}
}

func (s *ListState) animatedRequest(e *node, ensure func(int) bool, top, bottom, padTop, padBottom float64) (listRequest, bool) {
	m := s.motion
	if m == nil {
		return listRequest{}, false
	}
	if !ensure(m.row) {
		if s.motion == m {
			s.StopScrollAnimation()
		}
		return listRequest{}, false
	}
	if s.motion != m {
		return listRequest{}, false
	}
	hs := &s.heights
	size, regionTop := hs.height(m.row), 0.0
	if m.region {
		regionTop = min(m.top, size)
		size = min(m.height, size-regionTop)
	}
	shown, start := top, padTop
	if s.Header != nil && !s.Header(m.row) {
		if hh := s.headerAbove(m.row, e.list.n); hh >= 0 && ensure(hh) {
			if s.motion != m {
				return listRequest{}, false
			}
			shown = top + hs.height(hh)
			start = max(start, shown)
		}
	}
	if s.motion != m {
		return listRequest{}, false
	}
	y := start
	switch m.align {
	case Center:
		y = (shown + bottom - size) / 2
	case End:
		y = float64(e.h) - padBottom - size
	}
	content := max(padTop+padBottom+hs.top(e.list.n, s.gap)-s.gap, float64(e.h))
	goal := max(0, min(padTop+hs.top(m.row, s.gap)+regionTop-y, content-float64(e.h)))
	// Measurement can change the estimated total. Keep the currently displayed
	// anchor fixed while moving both ends of the animation's coordinate system.
	current := max(0, min(hs.top(s.anchor, s.gap)+s.inset, content-float64(e.h)))
	if !m.initialized {
		m.from, m.last, m.started, m.initialized = current, current, e.c.now, true
		m.initialFrame = e.c.rt.frame
		m.initialAnchor, m.initialInset = s.anchor, s.inset
	} else if m.initialFrame == e.c.rt.frame {
		m.from = max(0, min(hs.top(m.initialAnchor, s.gap)+m.initialInset, content-float64(e.h)))
	} else {
		m.from += current - m.last
	}
	// A parent's size transition can replace natural geometry with its
	// displayed geometry later in this first frame. Duration follows that
	// final distance, then stays fixed for the remainder of the movement.
	if m.initialFrame == e.c.rt.frame && m.spec.Duration != nil {
		m.duration = m.spec.Duration(float32(math.Abs(goal - m.from)))
	}
	progress := float32(1)
	if !e.c.rt.preferences().ReduceMotion && m.duration > 0 {
		progress = max(0, min(1, float32(e.c.now.Sub(m.started))/float32(m.duration)))
	}
	fraction := progress
	if m.spec.Ease != nil {
		fraction = m.spec.Ease(progress)
	}
	offset := m.from + (goal-m.from)*float64(fraction)
	if progress >= 1 {
		// Parent size transitions can lay the List out again in this frame.
		// Keep exact target alignment through the final layout, then retire it.
		m.completedFrame = e.c.rt.frame
		if m.region {
			// Place from the measured owner so newly measured neighboring rows
			// preserve the region's alignment as the estimated offsets change.
			return listRequest{set: true, offset: true, hint: true, row: m.row, y: goal}, true
		}
		return listRequest{set: true, row: m.row, align: m.align}, true
	} else {
		m.completedFrame = 0
		e.c.rt.animating = true
	}
	m.last = offset
	return listRequest{set: true, offset: true, y: offset}, true
}

package chat

import (
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// Dock motion tokens matching docs/motion.md, docs/features/composer-dock-stability.md,
// TaskDock.module.css, GoalBar.module.css, and QueueDock.tsx/QueueDock.module.css.
const (
	DockSlideDuration        = widgets.DefaultDuration // 200ms clipped vertical slide
	PlanLeavingMaxDuration   = 8000 * time.Millisecond
	PlanLeavingGraceDuration = 1000 * time.Millisecond
	SystemRowSettleDuration  = widgets.FlipDuration // 400ms SYSTEM_ROW_SETTLE_MS
)

type dockSlideAnim struct {
	initialized bool
	lastOpen    bool
	from        float32
	startedAt   time.Time
}

// DockSlideProgress animates a bottom dock panel between hidden (0 -> translateY(100%))
// and visible (1 -> translateY(0)) over DockSlideDuration (200ms) using DesignEase.
// First render settles immediately (no mount flash unless toggled later).
// If immediateClose is true (e.g., TaskDock collapsing on send), closing jumps to 0 at once
// matching TaskDock.module.css (`transition: none` on closed track).
func DockSlideProgress(c *ui.Context, host ui.Element, open bool, immediateClose bool) float32 {
	target := float32(0)
	if open {
		target = 1
	}
	st := ui.Local(host, "chat.dock.slide", func() dockSlideAnim { return dockSlideAnim{} })
	if !st.initialized {
		st.initialized = true
		st.lastOpen = open
		st.from = target
		return target
	}
	if c.Preferences().ReduceMotion || (!open && immediateClose) {
		st.lastOpen = open
		st.from = target
		st.startedAt = time.Time{}
		return target
	}
	now := c.Now()
	if now.IsZero() {
		st.lastOpen = open
		st.from = target
		st.startedAt = time.Time{}
		return target
	}
	cur := evalDockSlide(st, now)
	if open != st.lastOpen {
		st.from = cur
		st.lastOpen = open
		st.startedAt = now
	}
	p := evalDockSlide(st, now)
	if !st.startedAt.IsZero() && now.Sub(st.startedAt) < DockSlideDuration {
		c.AnimationFrame()
	}
	return p
}

func evalDockSlide(st *dockSlideAnim, now time.Time) float32 {
	if st.startedAt.IsZero() {
		return float32FromBool(st.lastOpen)
	}
	elapsed := now.Sub(st.startedAt)
	if elapsed <= 0 {
		return st.from
	}
	target := float32FromBool(st.lastOpen)
	if elapsed >= DockSlideDuration {
		return target
	}
	t := float32(elapsed) / float32(DockSlideDuration)
	return st.from + (target-st.from)*widgets.DesignEase(t)
}

func float32FromBool(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// PlanLeavingTracker models usePlanLeavingOnSend from TodoPanel.tsx:
// when a user sends a prompt (pendingSubmission), the plan dock hides immediately
// instead of blocking composer movement, holding up to 8s max or 1s after the
// local pending flag clears before showing again unless a fresh plan arrival occurs.
type PlanLeavingTracker struct {
	Leaving           bool
	PendingSince      time.Time
	GraceUntil        time.Time
	WasPending        bool
	LastPlanSignature string
}

// Update advances plan leaving state for one frame and returns whether the plan dock
// is suppressed while leaving on send.
func (p *PlanLeavingTracker) Update(now time.Time, hasPending bool, planSignature string) bool {
	if planSignature != "" && planSignature != p.LastPlanSignature && p.LastPlanSignature != "" {
		// New plan update arrived after send: clear leaving grace early.
		p.Leaving = false
		p.PendingSince = time.Time{}
		p.GraceUntil = time.Time{}
	}
	p.LastPlanSignature = planSignature

	if hasPending {
		if !p.WasPending {
			p.Leaving = true
			p.PendingSince = now
			p.GraceUntil = time.Time{}
		}
		p.WasPending = true
		if p.Leaving && !p.PendingSince.IsZero() && !now.IsZero() && now.Sub(p.PendingSince) >= PlanLeavingMaxDuration {
			p.Leaving = false
			p.PendingSince = time.Time{}
			p.GraceUntil = time.Time{}
		}
		return p.Leaving
	}

	if p.WasPending {
		p.WasPending = false
		if p.Leaving && !now.IsZero() {
			p.GraceUntil = now.Add(PlanLeavingGraceDuration)
		} else {
			p.Leaving = false
		}
	}

	if p.Leaving && !p.GraceUntil.IsZero() && !now.IsZero() && !now.Before(p.GraceUntil) {
		p.Leaving = false
		p.PendingSince = time.Time{}
		p.GraceUntil = time.Time{}
	}
	return p.Leaving
}

// SystemRowSettleTracker models QueueDock's SYSTEM_ROW_SETTLE_MS (400ms):
// when a queue row leaves without a pending user submission (system dispatch /
// completion), wait 400ms before switching the shell between full and compact/hidden
// so transient system transitions do not flicker the composer height.
// Explicit user removal/promotion and pending submissions settle immediately.
type SystemRowSettleTracker struct {
	Settling    bool
	SettleUntil time.Time
	PrevCount   int
}

// Update advances QueueDock settle state. userTriggered or hasPending settles immediately.
func (s *SystemRowSettleTracker) Update(now time.Time, count int, userTriggered bool, hasPending bool) (effectiveHasRows bool, stillSettling bool) {
	if count > 0 {
		s.Settling = false
		s.SettleUntil = time.Time{}
		s.PrevCount = count
		return true, false
	}
	if s.PrevCount > 0 && count == 0 {
		s.PrevCount = 0
		if !userTriggered && !hasPending && !now.IsZero() {
			s.Settling = true
			s.SettleUntil = now.Add(SystemRowSettleDuration)
		} else {
			s.Settling = false
			s.SettleUntil = time.Time{}
		}
	}
	if s.Settling {
		if userTriggered || hasPending || now.IsZero() || !now.Before(s.SettleUntil) {
			s.Settling = false
			s.SettleUntil = time.Time{}
			return false, false
		}
		return true, true
	}
	return false, false
}

// paintDockSlide wraps a bottom composer dock in a clipped vertical track that
// slides over DockSlideDuration (200ms) with DesignEase, keeping outer glass
// material opaque while fading inner content and disabling interaction when closing.
func paintDockSlide(c *ui.Context, host ui.Element, progress float32, open bool, contentH *float32, build func(contentOpacity float32)) {
	if !open && progress <= 0 {
		return
	}
	host.ClipY().Disabled(!open).Inert(!open)
	if progress < 1 && *contentH > 0 {
		host.Height(progress * *contentH)
	}
	host.Children(func() {
		inner := ui.Column(c).FillWidth().MinWidth(0)
		inner.ObserveLayout(func(r ui.Rect) {
			if r.H > 0 && (open || *contentH == 0) {
				*contentH = r.H
			}
		})
		if progress < 1 && *contentH > 0 {
			inner.OffsetAfterLayout(func(ui.Rect) ui.Point {
				return ui.Point{Y: (1 - progress) * *contentH}
			})
		}
		inner.Children(func() {
			build(progress)
		})
	})
}

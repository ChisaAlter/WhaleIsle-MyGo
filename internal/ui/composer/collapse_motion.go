package composer

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// Scroll-driven composer collapse thresholds matching
// ui-conversation/src/client/skeleton/composer-scroll-collapse.ts.
const (
	ComposerExpandBottomThresholdPx   float32 = 24
	ComposerCollapseBottomThresholdPx float32 = 64
	ComposerUnpinScrollDeltaPx        float32 = 32
	ComposerCollapseDuration                  = widgets.DefaultDuration
)

// ComposerCollapseMode describes whether the active composer card is expanded,
// collapsed into a capsule pill, or pinned expanded after an explicit user click.
type ComposerCollapseMode uint8

const (
	ComposerCollapseExpanded ComposerCollapseMode = iota
	ComposerCollapseCollapsed
	ComposerCollapsePinnedExpanded
)

// ComposerCollapseState stores the scroll-collapse state machine snapshot.
type ComposerCollapseState struct {
	Mode            ComposerCollapseMode
	HasPinnedScroll bool
	PinnedScrollTop float32
}

// ComposerScrollSample is one vertical scrollport measurement of the transcript.
type ComposerScrollSample struct {
	ScrollTop        float32
	ScrollHeight     float32
	ClientHeight     float32
	PrevScrollTop    float32
	PrevScrollHeight float32
	FollowingTail    bool
}

// ComposerCollapseGuards controls when the composer may collapse into pill form.
type ComposerCollapseGuards struct {
	ActivePhase              bool
	HasOverlay               bool
	HasPendingInteraction    bool
	HasMultilineOrAttachment bool
	HasEditSession           bool
	HasOpenPopover           bool
}

// CanCollapseComposer returns whether the composer may collapse under current guards.
func CanCollapseComposer(g ComposerCollapseGuards) bool {
	return g.ActivePhase &&
		!g.HasOverlay &&
		!g.HasPendingInteraction &&
		!g.HasMultilineOrAttachment &&
		!g.HasEditSession &&
		!g.HasOpenPopover
}

// IsComposerCollapsed returns whether the composer should render as a collapsed pill.
func IsComposerCollapsed(s ComposerCollapseState, g ComposerCollapseGuards) bool {
	return s.Mode == ComposerCollapseCollapsed && CanCollapseComposer(g)
}

// PinComposerExpanded pins the composer expanded after an explicit user interaction
// while scrolled up in the conversation.
func PinComposerExpanded(scrollTop float32) ComposerCollapseState {
	return ComposerCollapseState{
		Mode:            ComposerCollapsePinnedExpanded,
		HasPinnedScroll: true,
		PinnedScrollTop: scrollTop,
	}
}

// ReduceComposerScroll advances the composer scroll-collapse state machine for
// one transcript scrollport sample.
func ReduceComposerScroll(state ComposerCollapseState, sample ComposerScrollSample, guards ComposerCollapseGuards) ComposerCollapseState {
	maxScroll := max(float32(0), sample.ScrollHeight-sample.ClientHeight)
	distanceFromBottom := max(float32(0), maxScroll-sample.ScrollTop)

	if maxScroll <= ComposerCollapseBottomThresholdPx || distanceFromBottom <= ComposerExpandBottomThresholdPx {
		if state.Mode == ComposerCollapseExpanded && !state.HasPinnedScroll {
			return state
		}
		return ComposerCollapseState{Mode: ComposerCollapseExpanded}
	}

	if !CanCollapseComposer(guards) {
		if state.Mode == ComposerCollapseCollapsed {
			return ComposerCollapseState{Mode: ComposerCollapseExpanded}
		}
		if state.Mode == ComposerCollapsePinnedExpanded {
			if state.HasPinnedScroll && state.PinnedScrollTop == sample.ScrollTop {
				return state
			}
			return PinComposerExpanded(sample.ScrollTop)
		}
		return state
	}

	scrolledUp := sample.ScrollTop < sample.PrevScrollTop-0.5
	contentGrew := sample.ScrollHeight > sample.PrevScrollHeight+0.5

	if state.Mode == ComposerCollapseExpanded &&
		distanceFromBottom >= ComposerCollapseBottomThresholdPx &&
		(scrolledUp || (!sample.FollowingTail && !contentGrew)) {
		return ComposerCollapseState{Mode: ComposerCollapseCollapsed}
	}

	if state.Mode == ComposerCollapsePinnedExpanded &&
		distanceFromBottom >= ComposerCollapseBottomThresholdPx &&
		!contentGrew &&
		state.HasPinnedScroll &&
		float32(math.Abs(float64(sample.ScrollTop-state.PinnedScrollTop))) >= ComposerUnpinScrollDeltaPx {
		return ComposerCollapseState{Mode: ComposerCollapseCollapsed}
	}

	return state
}

type collapseAnimState struct {
	initialized bool
	lastTarget  float32
	from        float32
	startedAt   time.Time
}

// CollapseProgress returns a 0..1 eased progress toward collapsed (1) or expanded (0)
// over ComposerCollapseDuration (200ms) with DesignEase, snapping immediately when
// reduced motion is active or the frame has no monotonic clock.
func CollapseProgress(c *ui.Context, host ui.Element, collapsed bool) float32 {
	target := float32(0)
	if collapsed {
		target = 1
	}
	st := ui.Local(host, "composer.collapse.anim", func() collapseAnimState { return collapseAnimState{} })
	if !st.initialized {
		st.initialized = true
		st.lastTarget = target
		st.from = target
		return target
	}
	if c.Preferences().ReduceMotion {
		st.lastTarget = target
		st.from = target
		st.startedAt = time.Time{}
		return target
	}
	now := c.Now()
	if now.IsZero() {
		st.lastTarget = target
		st.from = target
		st.startedAt = time.Time{}
		return target
	}
	current := evalCollapseProgress(st, now)
	if target != st.lastTarget {
		st.from = current
		st.lastTarget = target
		st.startedAt = now
	}
	progress := evalCollapseProgress(st, now)
	if st.startedAt != (time.Time{}) && now.Sub(st.startedAt) < ComposerCollapseDuration {
		c.AnimationFrame()
	}
	return progress
}

func evalCollapseProgress(st *collapseAnimState, now time.Time) float32 {
	if st.startedAt.IsZero() {
		return st.lastTarget
	}
	elapsed := now.Sub(st.startedAt)
	if elapsed <= 0 {
		return st.from
	}
	if elapsed >= ComposerCollapseDuration {
		return st.lastTarget
	}
	t := float32(elapsed) / float32(ComposerCollapseDuration)
	return st.from + (st.lastTarget-st.from)*widgets.DesignEase(t)
}

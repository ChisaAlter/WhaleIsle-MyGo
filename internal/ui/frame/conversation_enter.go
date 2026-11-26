package frame

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// First-send transition starts immediately on the commit leaving Hero using
// the shared slow token (300ms cubic-bezier(.4,0,.2,1)), while header and
// chat content swap in over the fast/swap token (100ms).
const conversationEnterDuration = widgets.SlowDuration

// The state belongs to the resident conversation surface. It owns no worker,
// timer callback, model request or global observer; an unmounted surface has
// nothing left running. Measurement/displacement must occur after layout and
// before painting: Bounds from the previous frame is not the new card top.
type conversationEnterState struct {
	session, structure string
	hero, initialized  bool
	measuredHero       bool
	heroTop            float32
	pending            bool
	running            bool
	holdTop, from      float32
	currentOffset      float32
	heldAt             time.Time
	startedAt          time.Time
	stack, scroller    ui.Rect
	measuredHold       bool
	swapping           bool
	swapStartedAt      time.Time
}

// conversationEnter is called once the stable conversation owner is built.
// structure names only structural/attribute changes (dock/takeover/row kinds,
// not reply text or draft text). signal additionally accepts actual scroll
// events. The renderer then calls measure with the actual unshifted composer
// card, stack and scrollport rectangles before applying its returned Y offset
// to the stack; displacement must preserve sibling layout and input geometry.
func conversationEnter(c *ui.Context, owner ui.Element, sessionID string, hero, promptAttempted bool, structure string) *conversationEnterState {
	return conversationEnterLazy(c, owner, sessionID, hero, promptAttempted, func() string { return structure })
}

func conversationEnterLazy(c *ui.Context, owner ui.Element, sessionID string, hero, promptAttempted bool, structure func() string) *conversationEnterState {
	s := ui.Local(owner, "conversation-first-send", func() conversationEnterState { return conversationEnterState{} })
	s.beginLazy(c.Now(), sessionID, hero, promptAttempted, structure, c.Preferences().ReduceMotion)
	if s.pending || s.running || s.swapping {
		c.AnimationFrame()
	}
	return s
}

func (s *conversationEnterState) beginLazy(now time.Time, sessionID string, hero, attempted bool, structure func() string, reduced bool) {
	if !s.initialized || s.session != sessionID {
		*s = conversationEnterState{session: sessionID, hero: hero, initialized: true}
		return
	}
	wasHero := s.hero
	s.hero = hero
	if hero {
		s.cancel()
		if !wasHero {
			s.measuredHero = false
		}
	} else if wasHero && attempted && !reduced {
		s.swapping = true
		s.swapStartedAt = now
		if s.measuredHero {
			s.pending = true
			s.holdTop = s.heroTop
			s.heldAt = now
			s.structure = structure()
			return
		}
	}
	if reduced {
		s.cancel()
		return
	}
	if s.pending {
		sig := structure()
		if s.structure != sig {
			s.structure = sig
		}
	}
}

func conversationEnterLegacy(c *ui.Context, owner ui.Element, sessionID string, hero, promptAttempted bool, structure string) *conversationEnterState {
	s := ui.Local(owner, "conversation-first-send", func() conversationEnterState { return conversationEnterState{} })
	s.begin(c.Now(), sessionID, hero, promptAttempted, structure, c.Preferences().ReduceMotion)
	if s.pending || s.running || s.swapping {
		c.AnimationFrame()
	}
	return s
}

func (s *conversationEnterState) begin(now time.Time, sessionID string, hero, attempted bool, structure string, reduced bool) {
	if !s.initialized || s.session != sessionID {
		*s = conversationEnterState{session: sessionID, structure: structure, hero: hero, initialized: true}
		return
	}
	wasHero := s.hero
	s.hero = hero
	if hero {
		s.cancel()
		if !wasHero {
			s.measuredHero = false
		}
	} else if wasHero && attempted && !reduced {
		s.swapping = true
		s.swapStartedAt = now
		if s.measuredHero {
			s.pending = true
			s.holdTop = s.heroTop
			s.heldAt = now
		}
	}
	if reduced {
		s.cancel()
	}
	if s.structure != structure {
		s.structure = structure
		s.signal(now)
	}
}

// measure receives current, pre-displacement layout. During Hero every layout
// refreshes the source card top, including draft/window resizing. On the first
// active commit leaving Hero, the 300ms glide starts immediately with zero hold.
func (s *conversationEnterState) measure(now time.Time, card, stack, scroller ui.Rect) float32 {
	if card.W <= 0 || card.H <= 0 {
		return s.offset(now)
	}
	if s.hero {
		s.heroTop, s.measuredHero = card.Y, true
		return 0
	}
	if s.pending {
		s.pending, s.running = false, true
		s.startedAt = now
		s.from = s.holdTop - card.Y
		s.currentOffset = s.from
		s.stack, s.scroller, s.measuredHold = stack, scroller, true
	} else if s.running {
		s.stack, s.scroller = stack, scroller
		if drift := card.Y + s.from - s.holdTop; math.Abs(float64(drift)) > .5 {
			s.from -= drift
		}
	}
	return s.offset(now)
}

func (s *conversationEnterState) signal(now time.Time) {}

func (s *conversationEnterState) offset(now time.Time) float32 {
	if s.pending && now.Sub(s.heldAt) >= conversationEnterDuration {
		s.cancel()
		return 0
	}
	if s.running {
		elapsed := now.Sub(s.startedAt)
		if elapsed >= conversationEnterDuration {
			s.cancel()
			return 0
		}
		progress := max(float32(0), float32(elapsed)/float32(conversationEnterDuration))
		s.currentOffset = s.from * (1 - widgets.DesignEase(progress))
	}
	return s.currentOffset
}

// swapOpacity returns the 100ms fade-in alpha for header and transcript content
// on the commit leaving Hero (composer-draft-transition.md:20).
func (s *conversationEnterState) swapOpacity(now time.Time, reduced bool) float32 {
	if reduced || !s.swapping || now.IsZero() {
		s.swapping = false
		return 1
	}
	elapsed := now.Sub(s.swapStartedAt)
	if elapsed >= widgets.SwapDuration {
		s.swapping = false
		return 1
	}
	if elapsed <= 0 {
		return 0
	}
	return widgets.DesignEase(float32(elapsed) / float32(widgets.SwapDuration))
}

// cancel is also the animation-cancel/end adapter. Hero and session changes
// call it before the next measurement; it leaves no offset/play-state behind.
func (s *conversationEnterState) cancel() {
	s.pending, s.running, s.swapping = false, false, false
	s.holdTop, s.from, s.currentOffset = 0, 0, 0
	s.heldAt, s.startedAt, s.swapStartedAt = time.Time{}, time.Time{}, time.Time{}
	s.stack, s.scroller, s.measuredHold = ui.Rect{}, ui.Rect{}, false
}

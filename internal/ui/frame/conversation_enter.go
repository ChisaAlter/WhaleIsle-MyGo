package frame

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Pinned e353e8e ConversationContent.tsx137–254 and
// ConversationRoot.module.css444: a finite first-send hold, then the same
// 300ms cubic-bezier(.4,0,.2,1) as ui-theme's slow/ease-in-out tokens.
const (
	conversationEnterQuiet    = 80 * time.Millisecond
	conversationEnterCap      = 450 * time.Millisecond
	conversationEnterDuration = 300 * time.Millisecond
)

// The state belongs to the resident conversation surface. It owns no worker,
// timer callback, model request or global observer; an unmounted surface has
// nothing left running. Measurement/displacement must occur after layout and
// before painting: Bounds from the previous frame is not the new card top.
type conversationEnterState struct {
	session, structure string
	hero, initialized  bool
	measuredHero       bool
	heroTop            float32
	pending, holding   bool
	running            bool
	holdTop, from      float32
	currentOffset      float32
	heldAt, quietAt    time.Time
	startedAt          time.Time
	stack, scroller    ui.Rect
	measuredHold       bool
}

// conversationEnter is called once the stable conversation owner is built.
// structure names only structural/attribute changes (dock/takeover/row kinds,
// not reply text or draft text). signal additionally accepts actual scroll
// events. The renderer then calls measure with the actual unshifted composer
// card, stack and scrollport rectangles before applying its returned Y offset
// to the stack; displacement must preserve sibling layout and input geometry.
func conversationEnter(c *ui.Context, owner ui.Element, sessionID string, hero, promptAttempted bool, structure string) *conversationEnterState {
	s := ui.Local(owner, "conversation-first-send", func() conversationEnterState { return conversationEnterState{} })
	s.begin(c.Now(), sessionID, hero, promptAttempted, structure, c.Preferences().ReduceMotion)
	if s.pending || s.holding || s.running {
		// Geometry can drift while held. Keep a bounded set of layout frames;
		// text-only rebuilds call measure but do not move the quiet deadline.
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
	} else if wasHero && attempted && s.measuredHero && !reduced {
		s.pending = true
		s.holdTop = s.heroTop
		s.heldAt, s.quietAt = now, now
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
// refreshes the source card top, including draft/window resizing. During hold
// it folds drift back into the offset; only actual resize/signals rearm quiet.
func (s *conversationEnterState) measure(now time.Time, card, stack, scroller ui.Rect) float32 {
	if card.W <= 0 || card.H <= 0 {
		return s.offset(now)
	}
	if s.hero {
		s.heroTop, s.measuredHero = card.Y, true
		return 0
	}
	if s.pending {
		s.pending, s.holding = false, true
		s.from = s.holdTop - card.Y
		s.currentOffset = s.from
		s.stack, s.scroller, s.measuredHold = stack, scroller, true
	} else if s.holding {
		if s.measuredHold && (stack.W != s.stack.W || stack.H != s.stack.H || scroller.W != s.scroller.W || scroller.H != s.scroller.H) {
			s.signal(now)
		}
		s.stack, s.scroller = stack, scroller
		if drift := card.Y + s.currentOffset - s.holdTop; math.Abs(float64(drift)) > .5 {
			s.currentOffset -= drift
			s.from = s.currentOffset
		}
	}
	return s.offset(now)
}

// signal corresponds to the original structural mutation/resize/scroll
// observer. Character-data updates deliberately do not call it. Once started,
// observers are detached: subsequent signals cannot restart/extend this glide.
func (s *conversationEnterState) signal(now time.Time) {
	if s.pending || s.holding {
		s.quietAt = now
	}
}

func (s *conversationEnterState) offset(now time.Time) float32 {
	// A removed/hidden card may never supply its first active measurement.
	// It must not leave the resident surface asking for frames indefinitely.
	if s.pending && now.Sub(s.heldAt) >= conversationEnterCap {
		s.cancel()
		return 0
	}
	if s.holding && (now.Sub(s.quietAt) >= conversationEnterQuiet || now.Sub(s.heldAt) >= conversationEnterCap) {
		s.holding, s.running = false, true
		s.startedAt = now
		s.from = s.currentOffset
	}
	if s.running {
		elapsed := now.Sub(s.startedAt)
		if elapsed >= conversationEnterDuration {
			s.cancel()
			return 0
		}
		progress := max(float32(0), float32(elapsed)/float32(conversationEnterDuration))
		s.currentOffset = s.from * (1 - conversationEnterEase(progress))
	}
	return s.currentOffset
}

// cancel is also the animation-cancel/end adapter. Hero and session changes
// call it before the next measurement; it leaves no offset/play-state behind.
func (s *conversationEnterState) cancel() {
	s.pending, s.holding, s.running = false, false, false
	s.holdTop, s.from, s.currentOffset = 0, 0, 0
	s.heldAt, s.quietAt, s.startedAt = time.Time{}, time.Time{}, time.Time{}
	s.stack, s.scroller, s.measuredHold = ui.Rect{}, ui.Rect{}, false
}

// ui.EaseInOut is a different cubic; solve the pinned CSS curve's x(t) for
// time, then evaluate y(t). The bounded bisection has sub-pixel precision for
// the dimensions this native conversation surface can occupy.
func conversationEnterEase(progress float32) float32 {
	if progress <= 0 || progress >= 1 {
		return min(float32(1), max(float32(0), progress))
	}
	lo, hi := float32(0), float32(1)
	for n := 0; n < 20; n++ {
		u := (lo + hi) / 2
		v := 1 - u
		x := 3*v*v*u*.4 + 3*v*u*u*.2 + u*u*u
		if x < progress {
			lo = u
		} else {
			hi = u
		}
	}
	u := (lo + hi) / 2
	return 3*(1-u)*u*u + u*u*u
}

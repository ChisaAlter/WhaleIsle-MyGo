package frame

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func enterRect(y float32) ui.Rect { return ui.Rect{X: 20, Y: y, W: 600, H: 180} }

func enteringFixture(now time.Time) conversationEnterState {
	var s conversationEnterState
	s.begin(now, "host:session", true, false, "hero", false)
	s.measure(now, enterRect(220), enterRect(180), ui.Rect{W: 800, H: 700})
	s.begin(now, "host:session", false, true, "active", false)
	s.measure(now, enterRect(520), enterRect(480), ui.Rect{W: 800, H: 700})
	return s
}

func TestConversationEnterStartsImmediatelyAndPinsDrift(t *testing.T) {
	now := time.Unix(1000, 0)
	s := enteringFixture(now)
	// Leaving Hero starts the 300ms glide immediately on the first active frame (zero timer hold).
	if got := s.offset(now); got != -300 || !s.running || s.pending || s.startedAt != now {
		t.Fatalf("first active card did not start immediately from Hero top: %v %+v", got, s)
	}
	// Layout drift adjusts the remaining origin without resetting startedAt.
	if offset := s.measure(now.Add(30*time.Millisecond), enterRect(540), enterRect(500), ui.Rect{W: 800, H: 700}); offset >= 0 || s.startedAt != now {
		t.Fatal("drift compensation reset startedAt or lost offset", offset, s.startedAt)
	}
	middle := s.offset(now.Add(150 * time.Millisecond))
	if math.Abs(float64(middle-(-320*(1-.77556)))) > .02 {
		t.Fatal("glide did not use the pinned CSS ease-in-out", middle)
	}
	if s.offset(now.Add(300*time.Millisecond)) != 0 || s.running || s.pending {
		t.Fatal("animation end retained offset/play state", s)
	}
	// A later active repaint cannot create another first-send animation.
	s.begin(now.Add(time.Second), "host:session", false, true, "active", false)
	if s.measure(now.Add(time.Second), enterRect(560), enterRect(520), ui.Rect{W: 800, H: 700}) != 0 || s.pending {
		t.Fatal("an ordinary active repaint restarted first-send")
	}
}

func TestConversationEnterSwapOpacity100msAndReducedMotion(t *testing.T) {
	now := time.Unix(1000, 0)
	var s conversationEnterState
	s.begin(now, "host:session", true, false, "hero", false)
	s.measure(now, enterRect(220), enterRect(180), ui.Rect{W: 800, H: 700})
	s.begin(now, "host:session", false, true, "active", false)
	if !s.swapping || s.swapOpacity(now, false) != 0 {
		t.Fatalf("expected 0 swap alpha at commit leaving hero: %+v", s)
	}
	mid := s.swapOpacity(now.Add(50*time.Millisecond), false)
	if mid <= 0.5 || mid >= 1 {
		t.Fatalf("mid swap alpha = %v, want DesignEase(0.5) in (0.5, 1)", mid)
	}
	if end := s.swapOpacity(now.Add(100*time.Millisecond), false); end != 1 || s.swapping {
		t.Fatalf("swap alpha at 100ms = %v, swapping=%v, want 1/false", end, s.swapping)
	}
	// Reduced motion skips swap fade immediately.
	s.begin(now, "host:session", true, false, "hero", false)
	s.begin(now, "host:session", false, true, "active", true)
	if s.swapOpacity(now, true) != 1 || s.swapping {
		t.Fatalf("reduced motion did not skip swap fade: %+v", s)
	}
}

func TestConversationEnterRequiresSameSessionHeroAndAttempt(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, tc := range []struct {
		name, session                    string
		previousHero, attempted, reduced bool
	}{
		{"history opens", "host:session", false, true, false},
		{"message without local send", "host:session", true, false, false},
		{"different session", "host:other", true, true, false},
		{"same id replacement Host", "new-host:session", true, true, false},
		{"reduce motion", "host:session", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s conversationEnterState
			s.begin(now, "host:session", tc.previousHero, false, "before", false)
			s.measure(now, enterRect(220), enterRect(180), ui.Rect{W: 800, H: 700})
			s.begin(now, tc.session, false, tc.attempted, "after", tc.reduced)
			if s.measure(now, enterRect(520), enterRect(480), ui.Rect{W: 800, H: 700}) != 0 || s.pending || s.running {
				t.Fatal("unrelated/reduced-motion navigation animated", s)
			}
		})
	}
	for _, mode := range []string{"hero", "session", "cancel", "reduced"} {
		t.Run("clear-"+mode, func(t *testing.T) {
			s := enteringFixture(now)
			switch mode {
			case "hero":
				s.begin(now, "host:session", true, true, "hero", false)
			case "session":
				s.begin(now, "host:other", false, true, "active", false)
			case "cancel":
				s.cancel()
			case "reduced":
				s.begin(now, "host:session", false, true, "active", true)
			}
			if s.offset(now) != 0 || s.pending || s.running || !s.startedAt.IsZero() {
				t.Fatal("retired animation retained offset/timeline", s)
			}
		})
	}
}

func TestConversationEnterHeroMeasurementAndStructureSignal(t *testing.T) {
	now := time.Unix(1000, 0)
	var s conversationEnterState
	s.begin(now, "host:session", true, false, "hero", false)
	s.measure(now, enterRect(200), enterRect(160), ui.Rect{W: 800, H: 700})
	s.measure(now, enterRect(260), enterRect(220), ui.Rect{W: 900, H: 750})
	s.begin(now, "host:session", false, true, "active", false)
	if s.measure(now, enterRect(520), enterRect(480), ui.Rect{W: 900, H: 750}) != -260 {
		t.Fatal("draft/window resize left a stale Hero source rectangle")
	}
	before := session.ViewState{SelectedID: "session", Messages: []session.Message{{ID: "reply", Kind: session.MessageKindText, Role: "assistant", Body: "a", Streaming: true}}}
	after := before
	after.Messages = append([]session.Message(nil), before.Messages...)
	after.Messages[0].Body = "a longer streamed reply"
	key := conversationEnterStructure(Model{}, before, conversationActive, false)
	if next := conversationEnterStructure(Model{}, after, conversationActive, false); next != key {
		t.Fatal("character-data edits became structural churn")
	}
}

func TestConversationEnterUnmountDoesNotResumeAnOldRun(t *testing.T) {
	mounted, hero := true, true
	var state *conversationEnterState
	armed := false
	tt := ui.NewTester(func(c *ui.Context) {
		if !mounted {
			ui.Text(c, "Another main page")
			return
		}
		owner := ui.Column(c.Key("resident-conversation")).Fill()
		state = conversationEnter(c, owner, "host:session", hero, !hero, "same rows")
		var card ui.Rect
		owner.OffsetAfterLayout(func(r ui.Rect) ui.Point {
			offset := state.measure(c.Now(), card, r, r)
			armed = armed || state.running
			return ui.Point{Y: offset}
		}).Children(func() {
			ui.Box(c).Height(30).ObserveLayout(func(r ui.Rect) { card = r })
		})
	}, 400, 300)
	hero = false
	tt.Frame()
	if !armed {
		t.Fatal("fixture never armed its same-session immediate first-send glide")
	}
	retired := state
	mounted = false
	tt.Frame()
	mounted = true
	tt.Frame()
	if state == retired || state.pending || state.running || state.currentOffset != 0 {
		t.Fatal("remount resumed state belonging to the disposed conversation owner")
	}
}

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

func TestConversationEnterPinsActualCardAndStartsAfterQuiet(t *testing.T) {
	now := time.Unix(1000, 0)
	s := enteringFixture(now)
	if got := s.offset(now); got != -300 || !s.holding || s.running {
		t.Fatalf("first active card was not held at the Hero card top: %v %+v", got, s)
	}
	// Layout drift is pinned even when no structural change rearms the hold.
	if offset := s.measure(now.Add(30*time.Millisecond), enterRect(540), enterRect(500), ui.Rect{W: 800, H: 700}); offset != -320 || s.quietAt != now {
		t.Fatal("pin did not compensate drift or incorrectly postponed quiet", offset, s.quietAt)
	}
	if s.offset(now.Add(79*time.Millisecond)) != -320 || !s.holding {
		t.Fatal("hold ended before 80ms of quiet")
	}
	if s.offset(now.Add(80*time.Millisecond)) != -320 || !s.running || s.holding {
		t.Fatal("glide did not begin at the quiet boundary")
	}
	middle := s.offset(now.Add(230 * time.Millisecond))
	if math.Abs(float64(middle-(-320*(1-.77556)))) > .02 {
		t.Fatal("glide did not use the pinned CSS ease-in-out", middle)
	}
	if s.offset(now.Add(380*time.Millisecond)) != 0 || s.running || s.holding || s.pending {
		t.Fatal("animation end retained offset/play state", s)
	}
	// A later active repaint cannot create another first-send animation.
	s.begin(now.Add(time.Second), "host:session", false, true, "active", false)
	if s.measure(now.Add(time.Second), enterRect(560), enterRect(520), ui.Rect{W: 800, H: 700}) != 0 || s.pending {
		t.Fatal("an ordinary active repaint restarted first-send")
	}
}

func TestConversationEnterChurnRearmsQuietButNeverItsCap(t *testing.T) {
	now := time.Unix(1000, 0)
	s := enteringFixture(now)
	for _, elapsed := range []time.Duration{70, 140, 210, 280, 350, 420} {
		at := now.Add(elapsed * time.Millisecond)
		s.signal(at) // Structural, attribute or actual scroll signal.
		if s.offset(at) != -300 || !s.holding {
			t.Fatal("churn released the card before quiet or cap", elapsed)
		}
	}
	if s.heldAt != now || s.quietAt != now.Add(420*time.Millisecond) {
		t.Fatal("rearm moved the hard cap or lost the last signal")
	}
	s.offset(now.Add(450 * time.Millisecond))
	if !s.running || s.holding || s.startedAt != now.Add(450*time.Millisecond) {
		t.Fatal("continuous structural churn exceeded the 450ms hold cap")
	}
	s.signal(now.Add(460 * time.Millisecond))
	if s.holding || s.startedAt != now.Add(450*time.Millisecond) {
		t.Fatal("a detached observer restarted a running animation")
	}
	if s.offset(now.Add(750*time.Millisecond)) != 0 || s.running {
		t.Fatal("capped hold did not reach its finite end")
	}
}

func TestConversationEnterResizeAndScrollRearmWithoutTextEdits(t *testing.T) {
	now := time.Unix(1000, 0)
	s := enteringFixture(now)
	stack, viewport := enterRect(480), ui.Rect{W: 800, H: 700}
	stack.H++
	s.measure(now.Add(60*time.Millisecond), enterRect(520), stack, viewport)
	if s.quietAt != now.Add(60*time.Millisecond) {
		t.Fatal("stack resize did not rearm quiet")
	}
	viewport.W++
	s.measure(now.Add(100*time.Millisecond), enterRect(520), stack, viewport)
	if s.quietAt != now.Add(100*time.Millisecond) {
		t.Fatal("scrollport resize did not rearm quiet")
	}
	scroll := conversationEnterScroll{}
	scroll.scroll.Y = 4
	scroll.signal(&s, now.Add(140*time.Millisecond))
	if s.quietAt != now.Add(140*time.Millisecond) {
		t.Fatal("actual scroll offset did not rearm quiet")
	}
	scroll.signal(&s, now.Add(160*time.Millisecond))
	if s.quietAt != now.Add(140*time.Millisecond) {
		t.Fatal("unchanged scroll state continuously postponed quiet")
	}
	s.offset(now.Add(220 * time.Millisecond))
	if !s.running {
		t.Fatal("quiet after resize/scroll did not start the glide")
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
			if s.measure(now, enterRect(520), enterRect(480), ui.Rect{W: 800, H: 700}) != 0 || s.pending || s.holding || s.running {
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
			if s.offset(now) != 0 || s.pending || s.holding || s.running || !s.startedAt.IsZero() {
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
	s.begin(now.Add(50*time.Millisecond), "host:session", false, true, key, false)
	quiet := s.quietAt
	s.begin(now.Add(100*time.Millisecond), "host:session", false, true, key, false)
	if s.quietAt != quiet {
		t.Fatal("text-only frames postponed quiet")
	}
	after.Messages = append(after.Messages, session.Message{ID: "new-row", Role: "assistant", Kind: session.MessageKindText})
	next := conversationEnterStructure(Model{}, after, conversationActive, false)
	if key == next {
		t.Fatal("a new row did not change the structural signal")
	}
	s.begin(now.Add(120*time.Millisecond), "host:session", false, true, next, false)
	if s.quietAt != now.Add(120*time.Millisecond) {
		t.Fatal("new structure did not rearm quiet")
	}
}

func TestConversationEnterUnmountDoesNotResumeAnOldHold(t *testing.T) {
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
			armed = armed || state.holding || state.running
			return ui.Point{Y: offset}
		}).Children(func() {
			ui.Box(c).Height(30).ObserveLayout(func(r ui.Rect) { card = r })
		})
	}, 400, 300)
	hero = false
	tt.Frame()
	if !armed {
		t.Fatal("fixture never armed its same-session first-send hold")
	}
	retired := state
	mounted = false
	tt.Frame()
	mounted = true
	tt.Frame()
	if state == retired || state.pending || state.holding || state.running || state.currentOffset != 0 {
		t.Fatal("remount resumed state belonging to the disposed conversation owner")
	}
}

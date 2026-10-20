package widgets

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestStateDotPaintsItsState(t *testing.T) {
	for _, state := range []State{StateIdle, StateRunning, StateDone, StateWarning, StateError} {
		tt, theme := newTester(t, func(c *ui.Context) {
			ui.Column(c).Fill().Center().Children(func() { StateDot(c, state) })
		}, 40, 40)
		want := StateColor(theme, state).Over(theme.Background)
		if got := at(tt.Image(), 20, 20); !near(got, rgb(want), 2) {
			t.Errorf("%v: dot is %v, want %v", state, got, rgb(want))
		}
	}
}

func TestStateDotReservesTenDIPs(t *testing.T) {
	var bounds ui.Rect
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Center().Children(func() {
			dot := StateDot(c, StateDone)
			bounds = dot.Bounds()
		})
	}, 40, 40)
	tt.Frame()
	if bounds.W != StateDotSlot || bounds.H != StateDotSlot {
		t.Fatalf("dot slot is %vx%v, want %v", bounds.W, bounds.H, StateDotSlot)
	}
}

func TestStateDotRunningBreathes(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Center().Children(func() { StateDot(c, StateRunning) })
	}, 40, 40)
	base := time.Unix(0, 0)
	tt.SetTime(base)
	first := capture(tt.Image())
	tt.SetTime(base.Add(700 * time.Millisecond))
	second := capture(tt.Image())
	if sameFrame(first, second) {
		t.Fatal("the running ring does not breathe")
	}
}

func TestStateDotStillWithReduceMotion(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Center().Children(func() { StateDot(c, StateRunning) })
	}, 40, 40)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	base := time.Unix(0, 0)
	tt.SetTime(base)
	first := capture(tt.Image())
	tt.SetTime(base.Add(700 * time.Millisecond))
	second := capture(tt.Image())
	if !sameFrame(first, second) {
		t.Fatal("the ring still breathes with reduce motion")
	}
}

func TestStateColorDistinguishesStates(t *testing.T) {
	theme := ui.LightTheme()
	seen := map[ui.Color]State{}
	for _, state := range []State{StateIdle, StateRunning, StateDone, StateWarning, StateError} {
		c := StateColor(theme, state)
		if c.A == 0 {
			t.Errorf("%v has no color", state)
		}
		if other, ok := seen[c]; ok {
			t.Errorf("%v and %v share a color", state, other)
		}
		seen[c] = state
	}
	if got := StateColor(theme, StateIdle).Over(theme.Background); ContrastRatio(got, theme.Background) < 1.4 {
		t.Errorf("the idle dot disappears into the background: %.2f", ContrastRatio(got, theme.Background))
	}
}

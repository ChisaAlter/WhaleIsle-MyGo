package widgets

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// switchTest builds one switch at (20, 20) in a 200×100 window.
func switchTest(t *testing.T, on *bool) (*ui.Tester, *ui.Theme) {
	t.Helper()
	return newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			Switch(c, on).Label("Wrap lines")
		})
	}, 200, 100)
}

func TestSwitchIsThirtySixByTwenty(t *testing.T) {
	on := false
	tt, _ := switchTest(t, &on)
	r, ok := tt.Find("Wrap lines")
	if !ok {
		t.Fatal("the switch has no name for assistive technology")
	}
	if r.W != SwitchWidth || r.H != SwitchHeight {
		t.Fatalf("the switch is %v×%v, want %v×%v", r.W, r.H, SwitchWidth, SwitchHeight)
	}
}

func TestSwitchFlipsByClickAndSpace(t *testing.T) {
	on := false
	tt, _ := switchTest(t, &on)
	if err := tt.Click("Wrap lines"); err != nil {
		t.Fatal(err)
	}
	if !on {
		t.Fatal("the click did not turn the switch on")
	}
	if !tt.Focused("Wrap lines") {
		t.Fatal("the click did not focus the switch")
	}
	tt.Key(0, ui.KeySpace)
	if on {
		t.Fatal("Space did not turn the switch off")
	}
	tt.Key(0, ui.KeyEnter)
	if !on {
		t.Fatal("Enter did not turn the switch on")
	}
}

func TestSwitchShowsItsState(t *testing.T) {
	on := false
	tt, theme := switchTest(t, &on)
	base := time.Unix(0, 0)
	tt.SetTime(base)
	off := theme.Border.Mix(theme.TextMuted, 0.7)
	// The thumb rests on the left, so the right of the track shows the track.
	if got := at(tt.Image(), 50, 30); !near(got, rgb(off), 2) {
		t.Fatalf("the off track is %v, want %v", got, rgb(off))
	}
	tt.Click("Wrap lines")
	tt.SetTime(base.Add(300 * time.Millisecond))
	if got := at(tt.Image(), 24, 30); !near(got, rgb(theme.Accent), 2) {
		t.Fatalf("the on track is %v, want the accent %v", got, rgb(theme.Accent))
	}
	// The thumb slid to the right end: 36 wide, 2 of padding, a 16 thumb.
	if got := at(tt.Image(), 46, 30); !near(got, rgb(ui.RGB(255, 255, 255)), 2) {
		t.Fatalf("the thumb is not at the right: %v", got)
	}
}

func TestSwitchJumpsWithReduceMotion(t *testing.T) {
	on := false
	tt, theme := switchTest(t, &on)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	base := time.Unix(0, 0)
	tt.SetTime(base)
	tt.Click("Wrap lines")
	if got := at(tt.Image(), 24, 30); !near(got, rgb(theme.Accent), 2) {
		t.Fatalf("with reduce motion the thumb still slides: %v", got)
	}
}

func TestSwitchShowsAFocusRing(t *testing.T) {
	on := false
	tt, _ := switchTest(t, &on)
	tt.Move(200, 90)
	before := capture(tt.Image())
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Wrap lines") {
		t.Fatal("Tab did not reach the switch")
	}
	if after := capture(tt.Image()); sameFrame(before, after) {
		t.Fatal("the keyboard focus shows no ring")
	}
}

func TestSwitchWithoutStateIsInert(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			Switch(c, nil).Label("Inert")
		})
	}, 200, 100)
	tt.Click("Inert")
	if r, ok := tt.Find("Inert"); !ok || r.W != SwitchWidth {
		t.Fatalf("the nil switch is %v", r)
	}
}

func TestSwitchRowFlipsItsSetting(t *testing.T) {
	on := false
	var told []bool
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			SwitchRow(c, SwitchRowOptions{
				Title:       "Close to tray",
				Description: "Keeps the session running",
				On:          &on,
				Last:        true,
				OnChange:    func(on bool) { told = append(told, on) },
			})
		})
	}, 600, 200)
	title, ok := tt.Find("Close to tray")
	if !ok {
		t.Fatal("the row is missing")
	}
	// The switch sits in the right column, level with the row's text.
	tt.ClickAt(600-180+18, title.Y+22)
	if !on {
		t.Fatal("clicking the switch did not flip the setting")
	}
	if len(told) != 1 || !told[0] {
		t.Fatalf("told %v, want one turn on", told)
	}
}

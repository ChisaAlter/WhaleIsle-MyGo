package settings

import (
	"image/color"
	"testing"
	"time"

	"whaleisle/internal/ui/chrome"

	"github.com/egoist/mygo/ui"
)

// TestSettingsSectionSwapRidesTheSwapToken pins the section-switch fade to the
// original's dsh-motion-swap-in keyframe: opacity-only, the fast 100 ms token,
// reduced-motion off. Reveal is the helper settingsBody applies to the keyed
// options pane.
func TestSettingsSectionSwapRidesTheSwapToken(t *testing.T) {
	active := "a"
	progress := float32(-1)
	tt := ui.NewTester(func(c *ui.Context) {
		pane := ui.Column(c.Key("pane-" + active)).Fill()
		progress = chrome.Reveal(c, pane, chrome.SwapDuration)
		pane.Opacity(progress)
	}, 200, 100)
	t0 := time.Now()
	tt.SetTime(t0)
	tt.Frame()
	if progress < 0 || progress >= 1 {
		t.Fatalf("first frame should be mid-fade, got %v", progress)
	}
	// Section switch: a new keyed pane restarts the fade from zero.
	active = "b"
	tt.SetTime(t0.Add(2 * time.Second))
	tt.Frame()
	if progress >= 1 {
		t.Fatalf("section switch should re-enter at %v, got %v", 0.0, progress)
	}
	tt.SetTime(t0.Add(2*time.Second + 99*time.Millisecond))
	tt.Frame()
	if progress >= 1 {
		t.Fatalf("fade should still run at 99 ms, got %v", progress)
	}
	tt.SetTime(t0.Add(2*time.Second + 100*time.Millisecond))
	tt.Frame()
	if progress != 1 {
		t.Fatalf("fade should settle by the 100 ms swap token, got %v", progress)
	}
}

// TestSettingsSectionSwapReducedMotionSkipsTheFade pins the reduced-motion
// contract: the keyed content appears fully on the first frame.
func TestSettingsSectionSwapReducedMotionSkipsTheFade(t *testing.T) {
	progress := float32(-1)
	tt := ui.NewTester(func(c *ui.Context) {
		pane := ui.Column(c.Key("pane")).Fill()
		progress = chrome.Reveal(c, pane, chrome.SwapDuration)
		pane.Opacity(progress)
	}, 200, 100)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTime(time.Now())
	tt.Frame()
	if progress != 1 {
		t.Fatalf("reduced motion should skip the swap, got %v", progress)
	}
}

// TestSettingsHitAlphaTracksTheKeyframe pins the pulse curve: full accent
// through the 40% hold, the ease-out ramp to transparent, static under reduced
// motion.
func TestSettingsHitAlphaTracksTheKeyframe(t *testing.T) {
	start := time.Now()
	if a := settingsHitAlpha(start, start, false); a != 1 {
		t.Fatalf("hold should open at full fill, got %v", a)
	}
	if a := settingsHitAlpha(start.Add(settingsHitHold), start, false); a != 1 {
		t.Fatalf("fill should hold through 40%%, got %v", a)
	}
	mid := settingsHitAlpha(start.Add(settingsHitHold+(settingsHitDuration-settingsHitHold)/2), start, false)
	if mid <= 0 || mid >= 1 {
		t.Fatalf("fade should be partway, got %v", mid)
	}
	if a := settingsHitAlpha(start.Add(settingsHitDuration-time.Millisecond), start, false); a > .1 {
		t.Fatalf("fade should be nearly clear at the end, got %v", a)
	}
	if a := settingsHitAlpha(start.Add(settingsHitDuration), start, false); a != 0 {
		t.Fatalf("fade should end transparent, got %v", a)
	}
	if a := settingsHitAlpha(start.Add(settingsHitDuration/2), start, true); a != 1 {
		t.Fatalf("reduced motion keeps the static fill, got %v", a)
	}
}

// TestSettingsSearchHitPulsesThenClears drives the real row target: the jump
// fills the row with the accent wash, the fill is gone once the keyframe ends,
// and with it the repaints the pulse asked for.
func TestSettingsSearchHitPulsesThenClears(t *testing.T) {
	accent := ui.RGB(10, 20, 230)
	armed := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			row := ui.Column(c.Key("row-hit")).FillWidth().Height(40)
			if !armed {
				jump := settingsSearchJumpState(c)
				jump.Target = "row-hit"
				jump.Pending = true
				jump.HitFill = accent
				armed = true
			}
			settingsSearchTarget(c, "row-hit", "", row)
			row.Children(func() {})
		})
	}, 200, 60)
	t0 := time.Now()
	tt.SetTime(t0)
	tt.Frame()
	px := func() color.RGBA { return tt.Image().RGBAAt(10, 20) }
	if got := px(); got != (color.RGBA{R: 10, G: 20, B: 230, A: 255}) {
		t.Fatalf("hit should fill with the accent during the hold, got %v", got)
	}
	// Mid-fade the fill blends toward the background instead of snapping off.
	tt.SetTime(t0.Add(settingsHitHold + 480*time.Millisecond))
	tt.Frame()
	if got := px(); got == (color.RGBA{R: 10, G: 20, B: 230, A: 255}) || got == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("mid-fade should blend, got %v", got)
	}
	// Once the keyframe ends the row is plain again and nothing keeps asking
	// for frames.
	tt.SetTime(t0.Add(settingsHitDuration + 50*time.Millisecond))
	tt.Frame()
	if got := px(); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("hit should clear when the keyframe ends, got %v", got)
	}
}

// TestRailCellFocusRingOnlyOnKeyboardFocus pins the rail cell's focus-visible
// ring: the sidebar-ring shadow paints only while the cell carries keyboard
// focus.
func TestRailCellFocusRingOnlyOnKeyboardFocus(t *testing.T) {
	query := ""
	var ring ui.Color
	label := navigationLabel(Model{}, SectionGeneral)
	tt := ui.NewTester(func(c *ui.Context) {
		ring = tk(c).SidebarRing
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Padding(8).Children(func() {
			railButton(c, Model{}, navigationEntry{id: SectionGeneral, icon: "SettingsOutline"}, false, &query)
		})
	}, 200, 80)
	cell := buttonRect(tt.FindAll(label))
	if cell.W == 0 {
		t.Fatalf("rail cell not found: %v", tt.Texts())
	}
	edge := func() color.RGBA { return tt.Image().RGBAAt(int(cell.X)-1, int(cell.Y+cell.H/2)) }
	isRing := func(px color.RGBA) bool { return px.R == ring.R && px.G == ring.G && px.B == ring.B }
	if got := edge(); isRing(got) {
		t.Fatalf("no ring without focus, got %v", got)
	}
	// Pointer focus is not focus-visible.
	tt.ClickAt(cell.X+10, cell.Y+cell.H/2)
	if got := edge(); isRing(got) {
		t.Fatalf("pointer focus should not paint the ring, got %v", got)
	}
	tt.Key(0, ui.KeyTab)
	if got := edge(); !isRing(got) {
		t.Fatalf("keyboard focus should paint the sidebar-ring shadow, got %v want %v", got, ring)
	}
}

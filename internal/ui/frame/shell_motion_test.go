package frame

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
	"time"
)

func TestShellTrackMotionUsesLayoutAndPreservesPreferences(t *testing.T) {
	if shellTrackEase(0) != 0 || shellTrackEase(1) != 1 || math.Abs(float64(shellTrackEase(.5)-.77556)) > .001 {
		t.Fatal("source cubic timing curve mismatch")
	}
	sizes := DefaultLayout()
	sizes.Bottom = 0
	var drawn, observed ui.Rect
	var samples []float32
	tt := ui.NewTester(func(c *ui.Context) {
		sidebarColumns(c, &sizes, func() {
			seat := ui.Box(c).Key("native-bounds-probe").Fill()
			observed = seat.Bounds() // Same API as browserpreview.ViewDocked.
			seat.Draw(func(_ *ui.Painter, r ui.Rect) { drawn = r; samples = append(samples, r.W) })
		}, func() { ui.Box(c).Fill() })
	}, 1200, 800)
	initial := drawn.W
	sizes.ToggleSidebar()
	samples = nil
	tt.Frame()
	time.Sleep(60 * time.Millisecond)
	tt.Frame()
	moving := false
	for _, w := range samples {
		if w > SidebarCollapsedWidth()+.01 && w < initial-.01 {
			moving = true
		}
	}
	if !moving {
		t.Fatal("sidebar did not pass through intermediate real layout widths")
	}
	if sizes.Sidebar != 0 || sizes.SidebarExpanded != initial {
		t.Fatal("animation overwrote width preferences")
	}
	// Bounds used by native controllers follows the same actual animated box,
	// one frame behind by the toolkit contract; it is not a painter transform.
	if observed.W <= SidebarCollapsedWidth() || observed.W > initial || drawn.W > observed.W+.01 {
		t.Fatalf("native=%+v drawn=%+v", observed, drawn)
	}
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	tt.Frame()
	if drawn.W != SidebarCollapsedWidth() {
		t.Fatal("reduced motion did not settle actual layout", drawn)
	}
	sizes.ToggleSidebar()
	tt.Frame()
	if drawn.W != initial {
		t.Fatal("reduced-motion expand was not immediate", drawn)
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	sizes.ToggleSidebar()
	tt.Frame()
	tt.SetSize(1300, 800)
	if drawn.W != SidebarCollapsedWidth() {
		t.Fatal("window resize did not bypass easing", drawn)
	}

	// Closing resource/native content stops its callback immediately, even
	// while its empty track contracts. The terminal has the same input rule.
	sizes = DefaultLayout()
	details := float32(420)
	open := true
	rightBuilds, terminalBuilds := 0, 0
	var nativeRect ui.Rect
	tracks := ui.NewTester(func(c *ui.Context) {
		cols := sizes.ResolveColumns(1200, open)
		details = cols.Details
		sidebarColumns(c, &sizes, func() {}, func() {
			centerRightColumns(c, Model{Sizes: &sizes, FilePaneWidth: &details}, func() {
				centerTerminalColumn(c, Model{Sizes: &sizes, centerHeight: 800}, func() { ui.Box(c).Fill() }, func() { terminalBuilds++; ui.Box(c).Fill() })
			}, func() { rightBuilds++; node := ui.Box(c).Fill(); nativeRect = node.Bounds() })
		})
	}, 1200, 800)
	_ = nativeRect
	rememberedRight, rememberedMain := sizes.Right, sizes.Main
	rightBuilds, terminalBuilds = 0, 0
	open, sizes.Bottom = false, 0
	tracks.Frame()
	if rightBuilds != 0 || terminalBuilds != 0 {
		t.Fatal("closed native content retained input/render callbacks")
	}
	time.Sleep(310 * time.Millisecond)
	tracks.Frame()
	if sizes.Right != rememberedRight || sizes.Main != rememberedMain {
		t.Fatal("track animation mutated persisted dimensions")
	}
}

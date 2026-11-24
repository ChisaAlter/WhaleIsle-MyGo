package settings

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestSettingsOverflowScrollbarNeverShiftsTheLayout pins the overlay
// scrollbar's contract: whether the options pane overflows or not, content
// keeps the same horizontal box — scrolling moves the copy inside it, never
// the modal, the rail, or the content's width.
func TestSettingsOverflowScrollbarNeverShiftsTheLayout(t *testing.T) {
	height := float32(740) // taller than the options viewport
	section := SectionGeneral
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, Model{ActiveSection: section, OnClose: func() {}, SectionContent: func(c *ui.Context) {
			ui.Box(c).Label("Gutter content").FillWidth().Height(height)
		}})
	}, 1440, 864)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))
	box := func() ui.Rect {
		t.Helper()
		bounds, found := tt.Find("Gutter content")
		if !found {
			t.Fatal("content missing")
		}
		return bounds
	}
	sameBox := func(want ui.Rect) {
		t.Helper()
		if got := box(); math.Abs(float64(got.X-want.X)) > .01 || math.Abs(float64(got.W-want.W)) > .01 {
			t.Fatalf("layout shifted: got %+v want x=%g w=%g", got, want.X, want.W)
		}
	}
	wide := box()
	for frame := 0; frame < 6; frame++ {
		tt.Frame()
		sameBox(wide) // the gutter must not accumulate or oscillate
	}
	tt.Scroll(900, 400, 0, 12)
	if box().Y == wide.Y {
		t.Fatal("pane did not scroll; the overflow case is unproven")
	}
	sameBox(wide) // scrolling changes neither the content box nor the modal x
	height = 100  // fits entirely: the overlay scrollbar reserves nothing
	tt.Frame()
	sameBox(wide)
	height = 740
	tt.SetSize(600, 440)
	narrow := box()
	for frame := 0; frame < 4; frame++ {
		tt.Frame()
		sameBox(narrow)
	}
	height = 100
	tt.Frame()
	sameBox(narrow)
	section = SectionAppearance
	tt.Frame()
	sameBox(narrow) // each section owns its scroll state
	height = 600
	tt.Frame()
	sameBox(narrow)
	tt.SetSize(1440, 864)
	sameBox(wide)
}

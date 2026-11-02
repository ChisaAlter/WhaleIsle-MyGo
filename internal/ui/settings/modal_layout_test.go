package settings

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSettingsOverflowGutterTracksLayoutWithoutMovingModal(t *testing.T) {
	height := float32(740) // options viewport is 746, plus its 24 DIP bottom inset
	section := SectionGeneral
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, Model{ActiveSection: section, OnClose: func() {}, SectionContent: func(c *ui.Context) {
			ui.Box(c).Label("Gutter content").FillWidth().Height(height)
		}})
	}, 1440, 864)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))
	check := func(left, right float32) ui.Rect {
		t.Helper()
		bounds, found := tt.Find("Gutter content")
		if !found || math.Abs(float64(bounds.X-left)) > .01 || math.Abs(float64(bounds.X+bounds.W-right)) > .01 {
			t.Fatalf("content=%+v found=%v want horizontal [%g,%g]", bounds, found, left, right)
		}
		return bounds
	}
	first := check(532, 1091) // 1120 panel edge - 24 padding - source scrollbar 5
	for frame := 0; frame < 6; frame++ {
		tt.Frame()
		if got := check(532, 1091); got.W != first.W {
			t.Fatal("overflow gutter kept accumulating or oscillating")
		}
	}
	tt.Scroll(900, 400, 0, 12)
	check(532, 1091) // scrolling changes neither gutter nor modal x
	height = 721     // 721 + 24 < 746, so no scrollbar is needed
	tt.Frame()
	check(532, 1096)
	height = 722 // exact fit must not count as overflow
	tt.Frame()
	check(532, 1096)
	tt.SetSize(600, 440)
	check(236, 547) // narrow: panel edge576 -24 -5, left24+188+24
	for frame := 0; frame < 4; frame++ {
		tt.Frame()
		check(236, 547)
	}
	height = 100
	tt.Frame()
	check(236, 552)
	section = SectionAppearance
	tt.Frame()
	check(236, 552) // each section owns its scroll and gutter state
	height = 600
	tt.Frame()
	check(236, 547)
	tt.SetSize(1440, 864)
	check(532, 1096)
}

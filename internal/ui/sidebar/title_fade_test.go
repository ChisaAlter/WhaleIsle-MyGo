package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestSessionTitleFadeStates(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		hover                              bool
		offset, content, cell, left, right float32
	}{
		{"rest", false, -40, 300, 100, 0, 0},
		{"unmeasured", true, 0, 300, 0, 0, 0},
		{"small overflow", true, 0, 108, 100, 0, 0},
		{"start", true, 0, 300, 100, 0, 8},
		{"moving", true, -40, 300, 100, 8, 8},
		{"end or reduced motion", true, -200, 300, 100, 8, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := titleFadeEdges(tc.hover, tc.offset, tc.content, tc.cell)
			if left != tc.left || right != tc.right {
				t.Fatalf("edges=%v/%v want=%v/%v", left, right, tc.left, tc.right)
			}
		})
	}
}

// The actual native title still owns one immutable lane and one text. Reduced
// motion settles immediately, pointer leave returns the same whole-opacity ink.
func TestSessionTitleFadeLeavesSameRestingPixels(t *testing.T) {
	const title = "Beginning original session title retained through a long middle and final tail"
	for _, reduced := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			row := ui.Row(c).Height(32).FillWidth().Clip()
			row.Children(func() { sessionTitleCell(c, row, "fade", title, ui.RGB(220, 20, 20)) })
		}, 200, 48)
		tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
		before := append([]byte(nil), tt.Image().Pix...)
		tt.Move(80, 16)
		tt.SetTime(time.Now().Add(30 * time.Second))
		tt.Frame()
		after := tt.Image()
		changed := false
		for i := range before {
			if before[i] != after.Pix[i] {
				changed = true
				break
			}
		}
		if !changed {
			t.Fatalf("long title did not reveal on hover reduced=%v", reduced)
		}
		tt.Move(230, 45)
		tt.Frame()
		for i, value := range before {
			if tt.Image().Pix[i] != value {
				t.Fatalf("pointer leave failed to restore resting pixel %d reduced=%v", i, reduced)
			}
		}
	}
}

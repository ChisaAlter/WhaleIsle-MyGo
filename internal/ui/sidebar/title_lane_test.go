package sidebar

import (
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A long title must paint and take the pointer only inside its stationary
// column. OffsetAfterLayout on the old Text alone moved both into the status.
func TestSessionTitleLaneClipsStatusAndKeepsPointerBounds(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		name := "crawl"
		if reduced {
			name = "reduced-motion"
		}
		t.Run(name, func(t *testing.T) {
			const label = "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"
			clicked, renamed := 0, 0
			tt := ui.NewTester(func(c *ui.Context) {
				row := ui.Row(c).FillWidth().Height(32).AlignItems(ui.Center).Clip().Background(ui.RGB(255, 255, 255))
				row.Children(func() {
					ui.Box(c).Label("Leading status").Width(16).Height(20).Shrink(0).Background(ui.RGB(0, 80, 200))
					lane := sessionTitleCell(c, row, "long", label, ui.RGB(230, 0, 0)).Label("Title lane")
					if lane.DoubleClicked() {
						renamed++
					} else if lane.Clicked() {
						clicked++
					}
					ui.Box(c).Label("Trailing action").Width(40).Height(20).Shrink(0).Background(ui.RGB(0, 80, 200))
				})
			}, 240, 48)
			tt.SetPreferences(ui.Preferences{ReduceMotion: reduced})
			before, ok := tt.Find("Title lane")
			if !ok || before.X != 16 || before.W != 184 || before.H <= 0 || before.H > 32 {
				t.Fatalf("stationary title lane=%+v found=%t", before, ok)
			}
			baseline := image.NewRGBA(tt.Image().Bounds())
			copy(baseline.Pix, tt.Image().Pix)
			tt.Move(80, 16)
			tt.SetTime(time.Now().Add(30 * time.Second))
			tt.Frame()
			after, ok := tt.Find("Title lane")
			if !ok || after != before {
				t.Fatalf("scroll moved title pointer lane: before=%+v after=%+v", before, after)
			}
			frame := tt.Image()
			ink := false
			for y := 0; y < 32; y++ {
				for x := 0; x < 240; x++ {
					pixel := frame.RGBAAt(x, y)
					if x < 16 || x >= 200 {
						if pixel != baseline.RGBAAt(x, y) {
							t.Fatalf("title painted outside lane at %d,%d", x, y)
						}
					} else if int(pixel.R) > int(pixel.G)+30 && int(pixel.R) > int(pixel.B)+30 {
						ink = true
					}
				}
			}
			if !ink {
				t.Fatal("cropping removed the entire title")
			}
			// Pixels outside the lane cannot inherit the displaced title's click.
			tt.ClickAt(8, 16)
			tt.ClickAt(8, 16)
			if clicked != 0 || renamed != 0 {
				t.Fatal("leading status activated title")
			}
			tt.ClickAt(after.X+after.W-8, 16)
			tt.ClickAt(after.X+after.W-8, 16)
			if clicked < 1 || renamed != 1 {
				t.Fatalf("stationary lane title clicks=%d rename=%d", clicked, renamed)
			}
		})
	}
}

package chat

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// rowRevealFrames is how many frames the reveal is sampled over: 13 frames of
// 16 ms pass the 200 ms it takes.
const rowRevealFrames = 13

// sampleRowHeight reads the height of the tool row as the place of the element
// below it, every 16 ms from t. With TOOL_ROW_EVIDENCE_DIR set it also leaves
// one PNG per frame, so a change to the reveal can be compared frame by frame.
func sampleRowHeight(t *testing.T, tt *ui.Tester, from time.Time, label string) []float32 {
	t.Helper()
	dir := os.Getenv("TOOL_ROW_EVIDENCE_DIR")
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	heights := make([]float32, rowRevealFrames+1)
	for k := range heights {
		tt.SetTime(from.Add(time.Duration(k) * 16 * time.Millisecond))
		flow, ok := tt.Find("following flow")
		if !ok {
			t.Fatalf("row lost the element below it at frame %d", k)
		}
		heights[k] = flow.Y - 20
		if dir != "" {
			writeGallery(t, dir, fmt.Sprintf("tool-row-reveal-%s-f%02d.png", label, k), tt)
		}
	}
	return heights
}

func nearHeight(t *testing.T, label string, got, want float32) {
	t.Helper()
	if diff := got - want; diff > .5 || diff < -.5 {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}

// The row opens and closes by its own height over one duration and one easing,
// so the rows below follow it instead of jumping when the body appears or goes.
// Sampled on a fixed clock, one direction is the other subtracted from the open
// height, and no frame takes the whole step.
func TestToolRowRevealMovesTheRowBelowOverTheSameTimeBothWays(t *testing.T) {
	info := ToolCardInfo{Name: "custom", Arguments: "argument", Output: "result", Status: "ok"}
	tt, _ := toolRowViewFixture(t, &info, 640, false)
	base := time.Unix(7800, 0)
	if flow, _ := tt.Find("following flow"); flow.Y-20 != 24 {
		t.Fatalf("closed row height %v, want the 24px seat", flow.Y-20)
	}
	tt.ClickAt(610, 32)
	opened := sampleRowHeight(t, tt, base, "open")
	openHeight := opened[len(opened)-1]
	if openHeight <= 24 {
		t.Fatalf("row opened to %v, no taller than its seat", openHeight)
	}
	nearHeight(t, "open starts at the seat", opened[0], 24)
	distinct := 0
	for k := 1; k < len(opened); k++ {
		if opened[k] < opened[k-1] {
			t.Fatalf("open went backwards at frame %d: %v after %v", k, opened[k], opened[k-1])
		}
		if opened[k] != opened[k-1] && opened[k] != openHeight {
			distinct++
		}
	}
	if distinct < 6 {
		t.Fatalf("open took %d frames between the seat and the body: %v", distinct, opened)
	}
	tt.ClickAt(610, 32)
	closed := sampleRowHeight(t, tt, base.Add(rowRevealFrames*16*time.Millisecond), "close")
	nearHeight(t, "close starts at the open height", closed[0], openHeight)
	for k := 1; k < len(closed); k++ {
		if closed[k] > closed[k-1] {
			t.Fatalf("close went backwards at frame %d: %v after %v", k, closed[k], closed[k-1])
		}
		if k < len(opened) {
			nearHeight(t, "close mirrors open at frame "+(time.Duration(k*16)*time.Millisecond).String(), closed[k], openHeight-opened[k]+24)
		}
	}
	nearHeight(t, "close ends at the seat", closed[len(closed)-1], 24)
}

// The desktop's reduced motion setting takes the reveal in no frames at all.
func TestToolRowRevealTakesNoFramesUnderReducedMotion(t *testing.T) {
	info := ToolCardInfo{Name: "custom", Arguments: "argument", Output: "result", Status: "ok"}
	tt, _ := toolRowViewFixture(t, &info, 640, false)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.ClickAt(610, 32)
	opened := sampleRowHeight(t, tt, time.Unix(7800, 0), "reduced")
	for k, height := range opened {
		nearHeight(t, "reduced open frame "+(time.Duration(k*16)*time.Millisecond).String(), height, opened[len(opened)-1])
	}
	tt.ClickAt(610, 32)
	if flow, _ := tt.Find("following flow"); flow.Y-20 != 24 {
		t.Fatalf("reduced close kept %v, want the 24px seat", flow.Y-20)
	}
}

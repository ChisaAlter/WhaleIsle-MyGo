package widgets

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestFlipTextAnimatesAndSettlesIdle(t *testing.T) {
	val := "Fast"
	start := time.Unix(1000, 0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Row(c).Width(200).Padding(8).Children(func() {
			FlipText(c, val, FlipTextOptions{Key: "chip", FontSize: 14, LineHeight: 20})
		})
	}, 220, 48)
	tt.SetTime(start)
	if !tt.HasText("Fast") {
		t.Fatal("initial flip text missing")
	}

	val = "Deep"
	tt.SetTime(start.Add(time.Millisecond))
	// Mid-flip should still build/animate and reach settled at 400ms.
	tt.SetTime(start.Add(time.Millisecond + FlipDuration/2))
	if !tt.HasText("Deep") {
		t.Fatal("accessible new text missing mid-flip")
	}

	tt.SetTime(start.Add(time.Millisecond + FlipDuration + time.Millisecond))
	builds = 0
	tt.Frame()
	if builds != 1 {
		t.Fatalf("settled flip text triggered %d builds, want 1 (0 Hz idle)", builds)
	}
}

func TestFlipTextReduceMotionSkipsAnimation(t *testing.T) {
	val := "Alpha"
	start := time.Unix(1000, 0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Row(c).Width(200).Padding(8).Children(func() {
			FlipText(c, val, FlipTextOptions{Key: "chip", FontSize: 14, LineHeight: 20})
		})
	}, 220, 48)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.SetTime(start)
	val = "Beta"
	builds = 0
	tt.SetTime(start.Add(time.Millisecond))
	if builds != 1 {
		t.Fatalf("reduce-motion flip triggered %d builds, want 1", builds)
	}
	if !tt.HasText("Beta") {
		t.Fatal("new text missing under reduce motion")
	}
}

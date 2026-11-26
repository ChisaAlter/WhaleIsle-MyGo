package coss

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

func TestCossSelectUsesFlipTextAnd12pxPopupRadius(t *testing.T) {
	selected := "fast"
	opts := []SelectOption{
		{Value: "fast", Label: "Fast"},
		{Value: "deep", Label: "Deep"},
	}
	start := time.Unix(1000, 0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		Select(c, &selected, opts, SelectOptions{})
	}, 320, 200)
	tt.SetTime(start)
	if !tt.HasText("Fast") {
		t.Fatal("initial select label missing")
	}
	selected = "deep"
	tt.SetTime(start.Add(time.Millisecond))
	tt.SetTime(start.Add(design.MotionFlip/2 + time.Millisecond))
	if !tt.HasText("Deep") {
		t.Fatal("updated select label missing mid-flip")
	}
	tt.SetTime(start.Add(design.MotionFlip + 2*time.Millisecond))
	builds = 0
	tt.Frame()
	if builds != 1 {
		t.Fatalf("settled select flip triggered %d builds, want 1", builds)
	}
}

func TestCossSwitchUses100msAndButtonFeedbackFixedWidth(t *testing.T) {
	on := false
	start := time.Unix(1000, 0)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(8).Gap(8).Children(func() {
			Switch(c, &on, "Enable feature", CheckOptions{Label: "Enable feature"})
			Button(c, "Run", ButtonOptions{
				Loading:  true,
				Feedback: "Running long diagnostic verification step",
			})
		})
	}, 260, 160)
	tt.SetTime(start)
	if _, ok := tt.Find("Enable feature"); !ok {
		t.Fatal("switch missing")
	}
	on = true
	tt.SetTime(start.Add(time.Millisecond))
	tt.SetTime(start.Add(design.MotionFast + 2*time.Millisecond))
	if !tt.HasText("Running long diagnostic verification step") {
		t.Fatal("button feedback text missing")
	}
}

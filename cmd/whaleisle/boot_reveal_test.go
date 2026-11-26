package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/boot"
)

func TestColdBootHandoffTriggersHorizonRevealAndStopsIdle(t *testing.T) {
	var s mainState
	s.coldLoading = true
	s.hadColdLoading = true
	s.lastColdDark = true
	s.lastColdFailed = false

	start := time.Unix(1000, 0)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ui.Box(c).Fill().Background(ui.RGB(255, 0, 0))
		s.paintBootReveal(c)
	}, 200, 100)
	tt.SetTime(start)

	// While coldLoading is still true, reveal must not start.
	if s.bootReveal.Active() {
		t.Fatal("reveal started while coldLoading was still true")
	}

	// Host/workspace becomes ready -> coldLoading flips to false.
	s.coldLoading = false
	tt.SetTime(start.Add(time.Millisecond))
	if !s.bootReveal.Active() {
		t.Fatal("reveal did not trigger on coldLoading handoff")
	}
	if got := tt.Image().RGBAAt(100, 30); got.R > 40 {
		t.Fatalf("sky curtain did not cover workbench on handoff frame: %+v", got)
	}

	tt.SetTime(start.Add(time.Millisecond + boot.RevealDuration))
	if s.bootReveal.Active() {
		t.Fatal("reveal stayed active past RevealDuration")
	}
	builds = 0
	tt.Frame()
	if builds != 1 {
		t.Fatalf("settled reveal caused %d builds, want 1", builds)
	}
}

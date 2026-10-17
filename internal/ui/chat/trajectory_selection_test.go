package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
)

func TestTrajectorySelectionMinimumClickRange(t *testing.T) {
	s := trajectoryRangeState{}
	tt := ui.NewTester(func(c *ui.Context) {
		paintTrajectoryRange(c, rangeFixtureRows(), &s, nil, ui.RGB(50, 80, 200), ui.RGB(80, 80, 80), ui.RGB(150, 150, 150))
	}, 800, 200)
	r, _ := tt.Find("Trajectory time range")
	tt.Press(r.X+r.W*.4, r.Y+r.H/2)
	tt.Release(r.X+r.W*.4, r.Y+r.H/2)
	if !s.active || math.Abs(s.end-s.start-150) > 1e-5 {
		t.Fatalf("click minimum range: %f..%f", s.start, s.end)
	}
}
func TestTrajectorySelectionNearestAndShortDrag(t *testing.T) {
	full := buildRecordedTrajectoryTimeline(rangeFixtureRows())
	if got := nearestTrajectoryRow(full, 250); got != "a" {
		t.Fatalf("stable tie %s", got)
	}
	if got := nearestTrajectoryRow(full, 280); got != "b" {
		t.Fatalf("nearest %s", got)
	}
	s := trajectoryRangeState{anchor: 390}
	s.commitRange(full, full, 385, false)
	if s.start != 250 || s.end != 400 || s.focusID != "" {
		t.Fatalf("bounded short drag %+v", s)
	}
	s.anchor = 250
	s.commitRange(full, full, 250, true)
	if s.focusID != "a" || s.start != 175 || s.end != 325 {
		t.Fatalf("click focus %+v", s)
	}
	s.viewport = trajectoryViewport{active: true, start: 200, end: 220}
	s.anchor = 210
	s.commitRange(full, s.viewport.domain(full), 211, false)
	if s.end-s.start != 20 {
		t.Fatalf("viewport minimum %+v", s)
	}
}
func TestTrajectorySelectionClickSpanClearsRange(t *testing.T) {
	s := trajectoryRangeState{}
	s.selectRange(100, 400)
	selected := ""
	tt := ui.NewTester(func(c *ui.Context) {
		paintTrajectoryOverview(c, buildRecordedTrajectoryTimeline(rangeFixtureRows()), &selected, ui.RGB(50, 80, 200), ui.RGB(80, 80, 80), ui.RGB(150, 150, 150), false, &s)
	}, 800, 200)
	tt.Click("Trajectory span b")
	if selected != "b" || s.active {
		t.Fatalf("direct record selected=%s active=%v", selected, s.active)
	}
}

package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
	"whaleisle/internal/store/session"
)

func rangeFixtureRows() []trajRow {
	return []trajRow{
		{ID: "a", Timing: &session.TrajectoryTiming{OperationID: "a", StartedAt: trajectoryTestTime(100), CompletedAt: trajectoryTestTime(200)}},
		{ID: "b", Timing: &session.TrajectoryTiming{OperationID: "b", StartedAt: trajectoryTestTime(300), CompletedAt: trajectoryTestTime(400)}},
		{ID: "b-result", Timing: &session.TrajectoryTiming{OperationID: "b", StartedAt: trajectoryTestTime(300), CompletedAt: trajectoryTestTime(400)}},
		{ID: "unknown", EventTime: 350},
	}
}
func TestTrajectoryRangeRecordedOverlapAndNoInventedTime(t *testing.T) {
	s := trajectoryRangeState{}
	s.selectRange(390, 310)
	matches := s.matches(rangeFixtureRows())
	if len(matches) != 2 || !matches["b"] || !matches["b-result"] || matches["unknown"] {
		t.Fatalf("matches %v", matches)
	}
	if s.start != 310 || s.end != 390 {
		t.Fatal("reverse drag not normalized")
	}
	s.selectRange(500, 600)
	s.matches(rangeFixtureRows())
	if s.active {
		t.Fatal("nonoverlapping timeline retained stale range")
	}
	s.selectRange(math.NaN(), 1)
	if s.active {
		t.Fatal("invalid coordinates accepted")
	}
}
func TestTrajectoryRangePointerDragEscapeAndReset(t *testing.T) {
	s := trajectoryRangeState{}
	tt := ui.NewTester(func(c *ui.Context) {
		paintTrajectoryRange(c, rangeFixtureRows(), &s, nil, ui.RGB(50, 80, 200), ui.RGB(80, 80, 80), ui.RGB(150, 150, 150))
	}, 600, 200)
	r, ok := tt.Find("Trajectory time range")
	if !ok {
		t.Fatal("range input absent")
	}
	tt.Press(r.X+r.W*.7, r.Y+r.H/2)
	tt.Move(r.X+r.W*.9, r.Y+r.H/2)
	tt.Release(r.X+r.W*.9, r.Y+r.H/2)
	if !s.active || s.dragging || !s.matches(rangeFixtureRows())["b"] {
		t.Fatalf("drag %+v", s)
	}
	tt.Key(0, ui.KeyEscape)
	if s.active {
		t.Fatal("Escape did not clear range")
	}
	s.selectRange(300, 400)
	tt.Move(0, 0)
	tt.Click(trajectoryText("清除范围", "Clear range"))
	if s.active {
		t.Fatal("reset did not clear range")
	}
}

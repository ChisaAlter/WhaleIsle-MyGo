package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
)

func TestTrajectoryViewportAnchorBoundsAndMinimum(t *testing.T) {
	full := trajectoryTimeline{Start: 100, End: 1100}
	v := trajectoryViewport{}
	v.zoom(full, .75, -500, false)
	d := v.domain(full)
	if !v.active || math.Abs((d.Start+.75*(d.End-d.Start))-850) > 1e-8 {
		t.Fatalf("anchor %+v", v)
	}
	v.pan(full, v.start, 100)
	if v.start != 100 {
		t.Fatalf("left clamp %+v", v)
	}
	v.pan(full, v.start, -100)
	if v.end != 1100 {
		t.Fatalf("right clamp %+v", v)
	}
	v.zoom(full, .5, -100000, false)
	if math.Abs(v.end-v.start-20) > 1e-8 {
		t.Fatalf("minimum %+v", v)
	}
	v.zoom(full, .5, 100000, false)
	if v.active {
		t.Fatal("zoom out did not reset")
	}
	v.zoom(trajectoryTimeline{End: 10}, .5, -100000, true)
	if v.end-v.start != 4 {
		t.Fatalf("sequence minimum %+v", v)
	}
}
func TestTrajectoryViewportSequenceAndAxisReset(t *testing.T) {
	rows := append(rangeFixtureRows(), trajRow{ID: "request", Kind: "trajectory-request"})
	s := trajectoryRangeState{}
	s.setSequence(true)
	model := s.model(rows)
	if model.Start != 0 || model.End != 4 || len(model.Spans) != 4 {
		t.Fatalf("sequence %+v", model)
	}
	s.selectRange(3.1, 3.9)
	if m := s.matches(rows); len(m) != 1 || !m["unknown"] {
		t.Fatalf("ordinal selection %v", m)
	}
	s.viewport = trajectoryViewport{start: 1, end: 3, active: true}
	s.clear()
	if !s.viewport.active || !s.sequence {
		t.Fatal("clearing selection cleared viewport")
	}
	s.setSequence(false)
	if s.active || s.viewport.active || s.sequence {
		t.Fatal("axis retained stale coordinates")
	}
}
func TestTrajectoryViewportWheelAndSequenceRange(t *testing.T) {
	rows := make([]trajRow, 10)
	for i := range rows {
		rows[i] = trajRow{ID: string(rune('a' + i))}
	}
	s := trajectoryRangeState{sequence: true}
	tt := ui.NewTester(func(c *ui.Context) {
		paintTrajectoryRange(c, rows, &s, nil, ui.RGB(50, 80, 200), ui.RGB(80, 80, 80), ui.RGB(150, 150, 150))
	}, 800, 200)
	r, ok := tt.Find("Trajectory time range")
	if !ok {
		t.Fatal("missing range")
	}
	tt.Scroll(r.X+r.W*.5, r.Y+r.H/2, 0, -400)
	if !s.viewport.active || s.viewport.end-s.viewport.start >= 10 {
		t.Fatalf("wheel did not zoom %+v", s.viewport)
	}
	r, ok = tt.Find("Trajectory time range")
	if !ok {
		t.Fatal("range lost after zoom")
	}
	tt.Press(r.X+r.W*.4, r.Y+r.H/2)
	tt.Move(r.X+r.W*.6, r.Y+r.H/2)
	tt.Release(r.X+r.W*.6, r.Y+r.H/2)
	if !s.active || len(s.matches(rows)) == 0 {
		t.Fatalf("sequence drag unavailable state=%+v rect=%+v", s, r)
	}
	tt.Click(trajectoryText("重置缩放", "Reset zoom"))
	if s.viewport.active {
		t.Fatal("reset zoom failed")
	}
	if !s.active {
		t.Fatal("zoom reset lost range")
	}
}

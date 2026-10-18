package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestTrajectoryProjectionDefaultCompressesIdle(t *testing.T) {
	rows := rangeFixtureRows()
	messages := []session.Message{}
	for _, r := range rows[:2] {
		messages = append(messages, session.Message{ID: r.ID, Kind: session.MessageKindText, Role: "assistant", Body: r.ID, TrajectoryTime: r.Timing})
	}
	actual := true
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{Messages: messages}, ActualDuration: &actual})
	}, 900, 600)
	if !tt.HasText("已记录时间范围 200 ms") {
		t.Fatalf("default duration is not compressed: %v", tt.Texts())
	}
}
func TestTrajectoryProjectionOverlapIdleAndImmutability(t *testing.T) {
	full := trajectoryTimeline{Start: 100, End: 1000, Spans: []trajectorySpan{
		{Row: trajRow{ID: "late"}, Start: 900, End: 1000},
		{Row: trajRow{ID: "long"}, Start: 100, End: 400},
		{Row: trajRow{ID: "parallel"}, Start: 200, End: 250},
		{Row: trajRow{ID: "unknown"}, Start: 600, End: 600},
	}}
	got := projectTrajectoryTimeline(full, true, false)
	if got.Start != 100 || got.End != 500 || got.Spans[0].Start != 400 || got.Spans[1].End != 400 || got.Spans[2].Start != 200 || got.Spans[3].Start != 400 {
		t.Fatalf("projection %+v", got)
	}
	if full.Spans[0].Start != 900 {
		t.Fatal("mutated historical coordinates")
	}
	if got.Spans[0].End-got.Spans[0].Start != 100 || got.Spans[3].End != got.Spans[3].Start {
		t.Fatal("invented duration")
	}
	points := projectTrajectoryTimeline(full, false, true)
	if points.Start != 100 || points.End != 900 || points.Spans[1].Start != points.Spans[1].End {
		t.Fatalf("complete time markers %+v", points)
	}
}
func TestTrajectoryProjectionFocusUsesWholeCompressedModel(t *testing.T) {
	rows := rangeFixtureRows()
	s := trajectoryRangeState{}
	s.setProjection(false, true, false)
	s.selectRange(225, 250)
	got := s.matches(rows)
	if len(got) != 2 || !got["b"] || !got["b-result"] || got["a"] {
		t.Fatalf("compressed focus %v", got)
	}
	s.viewport = trajectoryViewport{active: true, start: 100, end: 200}
	s.setProjection(false, false, false)
	if s.active || s.viewport.active {
		t.Fatal("axis change retained incompatible coordinates")
	}
	if s.model(rows).End != 400 {
		t.Fatal("complete time domain lost")
	}
}
func TestTrajectoryProjectionCompleteTimeOptIn(t *testing.T) {
	actual, complete := true, true
	rows := rangeFixtureRows()
	messages := []session.Message{}
	for _, r := range rows[:2] {
		messages = append(messages, session.Message{ID: r.ID, Kind: session.MessageKindText, Role: "assistant", Body: r.ID, TrajectoryTime: r.Timing})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{Messages: messages}, ActualDuration: &actual, ActualTime: &complete})
	}, 900, 600)
	if !tt.HasText("已记录时间范围 300 ms") || tt.HasText("空闲已压缩") {
		t.Fatalf("complete time %v", tt.Texts())
	}
}

package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestTrajectoryUnknownDoesNotClaimEventTimeAsDuration(t *testing.T) {
	rows := buildTrajectoryRows([]session.Message{{ID: "missing-start", Kind: session.MessageKindText, Role: "assistant", EventTime: 1760000000000}}, "", false, false)
	if rows[0].Time != "未知" {
		t.Fatalf("timestamp without start cannot be a duration: %q", rows[0].Time)
	}
}

func trajectoryTestTime(v float64) *float64 { return &v }

func TestTrajectoryTimelineUsesRecordedParallelCoordinates(t *testing.T) {
	rows := []trajRow{
		{ID: "a", Kind: session.MessageKindText, Timing: &session.TrajectoryTiming{OperationID: "a", StartedAt: trajectoryTestTime(1000), CompletedAt: trajectoryTestTime(2000)}},
		{ID: "t", Kind: session.MessageKindTool, Timing: &session.TrajectoryTiming{OperationID: "t", StartedAt: trajectoryTestTime(1500), CompletedAt: trajectoryTestTime(1900)}},
		{ID: "t-result", Kind: session.MessageKindTool, Timing: &session.TrajectoryTiming{OperationID: "t", StartedAt: trajectoryTestTime(1500), CompletedAt: trajectoryTestTime(1900)}},
		{ID: "live", Kind: session.MessageKindTool, Timing: &session.TrajectoryTiming{OperationID: "live", StartedAt: trajectoryTestTime(1800)}},
		{ID: "partial", Kind: session.MessageKindText, Timing: &session.TrajectoryTiming{OperationID: "partial", CompletedAt: trajectoryTestTime(2500), Closed: true}},
	}
	m := buildRecordedTrajectoryTimeline(rows)
	if len(m.Spans) != 3 || m.Start != 1000 || m.End != 2000 || m.Unknown != 2 {
		t.Fatalf("timeline=%+v", m)
	}
	if m.Spans[1].Start != 1500 || m.Spans[1].End != 1900 || m.Spans[1].Lane == m.Spans[0].Lane {
		t.Fatalf("parallel tool placement=%+v", m.Spans)
	}
	if m.Spans[2].DurationKnown || m.Spans[2].Start != m.Spans[2].End {
		t.Fatalf("live duration invented: %+v", m.Spans[2])
	}
	if got := trajectoryDurationLabel(rows[0].Timing, false); got != "1000 ms" {
		t.Fatal(got)
	}
}

func TestTrajectoryRecordedViewDetails(t *testing.T) {
	actual := true
	selected := "a"
	tab := "timing"
	timing := &session.TrajectoryTiming{OperationID: "assistant:1:1", StartedAt: trajectoryTestTime(1000), FirstTokenAt: trajectoryTestTime(1200), CompletedAt: trajectoryTestTime(2000), Closed: true}
	tester := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{Messages: []session.Message{{ID: "a", Kind: session.MessageKindText, Role: "assistant", Body: "answer", TrajectoryTime: timing, EventTime: 2000}}}, ActualDuration: &actual, SelectedID: &selected, DetailTab: &tab})
	}, 900, 600)
	texts := strings.Join(tester.Texts(), "\n")
	for _, want := range []string{"1000 ms", "首 token：200 ms", "解码：800 ms", "已记录时间范围 1000 ms"} {
		if !strings.Contains(texts, want) {
			t.Fatalf("missing %s in %s", want, texts)
		}
	}
}

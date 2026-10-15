package chat

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func trajectoryDurationLabel(t *session.TrajectoryTiming, streaming bool) string {
	if value, ok := t.DurationMillis(); ok {
		return fmt.Sprintf("%.0f ms", value)
	}
	if streaming {
		return i18n.T(i18n.KeyTrajInProgress) + " · " + trajectoryText("未知", "Unknown")
	}
	if t != nil && !t.Closed {
		return trajectoryText("未记录结束 · 未知", "No recorded end · Unknown")
	}
	return trajectoryText("未知", "Unknown")
}

func trajectoryTimingDetail(row trajRow) string {
	parts := []string{trajectoryText("耗时：", "Duration: ") + row.Time}
	if row.EventTime > 0 {
		parts = append(parts, trajectoryText("事件时间：", "Event time: ")+formatEventTime(row.EventTime))
	}
	t := row.Timing
	if t == nil {
		return strings.Join(parts, "\n")
	}
	if t.StartedAt != nil {
		parts = append(parts, trajectoryText("开始：", "Started: ")+formatEventTime(*t.StartedAt))
	}
	if t.CompletedAt != nil {
		parts = append(parts, trajectoryText("完成：", "Completed: ")+formatEventTime(*t.CompletedAt))
	}
	if t.Invalid {
		parts = append(parts, trajectoryText("记录冲突，耗时未知", "Conflicting records; duration unknown"))
		return strings.Join(parts, "\n")
	}
	if t.StartedAt != nil && t.FirstTokenAt != nil && *t.FirstTokenAt >= *t.StartedAt && (t.CompletedAt == nil || *t.CompletedAt >= *t.FirstTokenAt) {
		parts = append(parts, fmt.Sprintf(trajectoryText("首 token：%.0f ms", "First token: %.0f ms"), *t.FirstTokenAt-*t.StartedAt))
		if t.CompletedAt != nil {
			parts = append(parts, fmt.Sprintf(trajectoryText("解码：%.0f ms", "Decoding: %.0f ms"), *t.CompletedAt-*t.FirstTokenAt))
		}
	} else if strings.HasPrefix(t.OperationID, "assistant:") {
		parts = append(parts, trajectoryText("首 token：未知", "First token: Unknown"), trajectoryText("解码：未知", "Decoding: Unknown"))
	}
	return strings.Join(parts, "\n")
}

type trajectorySpan struct {
	Row           trajRow
	Start, End    float64
	Lane          int
	DurationKnown bool
}

type trajectoryTimeline struct {
	Spans      []trajectorySpan
	Start, End float64
	Unknown    int
}

// One span per recorded operation; parallel calls share wall-clock coordinates.
// Unknown durations are timestamp markers, not equal-weight duration blocks.
func buildRecordedTrajectoryTimeline(rows []trajRow) trajectoryTimeline {
	var model trajectoryTimeline
	seen := make(map[string]bool)
	for _, row := range rows {
		t := row.Timing
		if t == nil {
			if row.Role == "assistant" || row.Kind == session.MessageKindTool || row.Kind == session.MessageKindThinking || row.Kind == session.MessageKindCompaction {
				model.Unknown++
			}
			continue
		} // User/log timestamps are not operation durations.
		if seen[t.OperationID] {
			continue
		}
		seen[t.OperationID] = true
		duration, known := t.DurationMillis()
		if !known {
			model.Unknown++
		}
		if t.Invalid || t.StartedAt == nil {
			continue
		}
		start := *t.StartedAt
		if math.IsNaN(start) || math.IsInf(start, 0) {
			continue
		}
		end := start
		if known {
			end += duration
		}
		lane := 0
		if row.Kind == session.MessageKindTool {
			lane = 1
		}
		if row.Kind == session.MessageKindCompaction {
			lane = 2
		}
		if len(model.Spans) == 0 || start < model.Start {
			model.Start = start
		}
		if len(model.Spans) == 0 || end > model.End {
			model.End = end
		}
		model.Spans = append(model.Spans, trajectorySpan{Row: row, Start: start, End: end, Lane: lane, DurationKnown: known})
	}
	return model
}

func paintRecordedTrajectoryTimeline(c *ui.Context, rows []trajRow, selected *string, accent, muted, border ui.Color) {
	model := buildRecordedTrajectoryTimeline(rows)
	if len(model.Spans) == 0 {
		ui.Text(c, trajectoryText("暂无已记录的起止时间；耗时未知", "No recorded start/end times; durations unknown")).FontSize(11).TextColor(muted)
		return
	}
	ui.Box(c).FillWidth().Height(38).Children(func() {
		for _, span := range model.Spans {
			left, width := float32(0), float32(0)
			if model.End > model.Start {
				left = float32((span.Start - model.Start) / (model.End - model.Start) * 100)
				width = float32((span.End - span.Start) / (model.End - model.Start) * 100)
			}
			row := span.Row
			b := ui.Button(c, "").Absolute().LeftPercent(left).Top(float32(span.Lane * 12)).WidthPercent(width).Height(9).
				Padding(0).BorderWidth(0).Radius(2).Background(trajKindColor(row.Kind, accent, muted, border)).
				Tooltip(trajectoryTimingDetail(row))
			if width == 0 {
				b.Width(2)
			} // Marker only; the label still explicitly says unknown/0 ms.
			if b.Clicked() {
				*selected = row.ID
			}
		}
	})
	label := fmt.Sprintf(trajectoryText("已记录时间范围 %.0f ms", "Recorded time range %.0f ms"), model.End-model.Start)
	if model.Unknown > 0 {
		label += fmt.Sprintf(trajectoryText(" · %d 项耗时未知", " · %d durations unknown"), model.Unknown)
	}
	ui.Text(c, label).FontSize(11).TextColor(muted)
}

func trajectoryText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

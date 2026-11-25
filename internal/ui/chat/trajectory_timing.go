package chat

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	firstLine := TrajectoryLocale("timing.totalDuration") + "：" + row.Time
	if i18n.Locale() == i18n.LocaleEN {
		firstLine = TrajectoryLocale("timing.totalDuration") + ": " + row.Time + " (Duration: " + row.Time + ")"
	}
	parts := []string{firstLine}
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
		if t.CompletedAt != nil {
			ttftMs := *t.FirstTokenAt - *t.StartedAt
			decMs := *t.CompletedAt - *t.FirstTokenAt
			ttftStr := fmt.Sprintf("%.0f ms", ttftMs)
			decStr := fmt.Sprintf("%.0f ms", decMs)
			if i18n.Locale() == i18n.LocaleEN {
				parts = append(parts,
					TrajectoryLocale("timing.generation")+": "+FormatTrajectoryTimelineTTFTDecoding(ttftStr, decStr)+
						fmt.Sprintf(" (First token %.0f ms · decoding %.0f ms)", ttftMs, decMs),
				)
			} else {
				parts = append(parts,
					TrajectoryLocale("timing.generation")+"："+FormatTrajectoryTimelineTTFTDecoding(ttftStr, decStr),
				)
			}
		} else {
			parts = append(parts, fmt.Sprintf(trajectoryText("首 token %.0f ms", "First token %.0f ms"), *t.FirstTokenAt-*t.StartedAt))
		}
	} else if strings.HasPrefix(t.OperationID, "assistant:") {
		parts = append(parts, trajectoryText("首 token 时间不可用", "First token time unavailable"), trajectoryText("解码：未知", "Decoding: Unknown"))
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
	Compressed bool
}

// One span per recorded operation; parallel calls share wall-clock coordinates.
// Unknown durations are timestamp markers, not equal-weight duration blocks.
func buildRecordedTrajectoryTimeline(rows []trajRow) trajectoryTimeline {
	var model trajectoryTimeline
	seen := make(map[string]bool)
	for _, row := range rows {
		t := row.Timing
		if t == nil {
			// User/context records are timestamp markers on lane 0, not
			// operation durations (timeline.ts input lane).
			if (row.Role == "user" || row.Role == "context") && row.EventTime > 0 && !math.IsNaN(row.EventTime) && !math.IsInf(row.EventTime, 0) {
				if len(model.Spans) == 0 || row.EventTime < model.Start {
					model.Start = row.EventTime
				}
				if row.EventTime > model.End {
					model.End = row.EventTime
				}
				model.Spans = append(model.Spans, trajectorySpan{Row: row, Start: row.EventTime, End: row.EventTime, Lane: 0})
				continue
			}
			if row.Role == "assistant" || row.Kind == session.MessageKindTool || row.Kind == session.MessageKindThinking || row.Kind == session.MessageKindCompaction {
				model.Unknown++
			}
			continue
		}
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
		lane := trajectorySpanLane(row)
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

// trajectorySpanLane is timeline.ts laneFor: 0 输入 (input/context),
// 1 模型 (request/message/compacted), 2 工具 (tool/subtool).
func trajectorySpanLane(row trajRow) int {
	switch {
	case row.Kind == session.MessageKindTool:
		return 2
	case row.Kind == trajectoryRequestKind,
		row.Kind == session.MessageKindCompaction,
		row.Kind == session.MessageKindThinking,
		row.Role == "assistant":
		return 1
	}
	return 0
}

// paintTrajectoryLanes renders the original compact lane chart
// (TrajectoryTimeline.module.css): a 44px label gutter beside a 50px plot,
// lanes 输入 / 模型 / 工具 at 14px pitch with 8px bars of min width 2,
// colored by kind. The plot itself carries the drag-range / wheel-zoom /
// Escape interaction the source renders on its overview.
func paintTrajectoryLanes(c *ui.Context, rows []trajRow, selected *string, s *trajectoryRangeState, list *ui.ListState, ledger []trajRow, p trajectoryPalette) {
	full := s.model(rows)
	model := s.viewport.domain(full)
	s.matches(rows)
	lanes := []string{trajectoryText("输入", "Input"), trajectoryText("模型", "Model"), trajectoryText("工具", "Tool")}
	ui.Row(c).FillWidth().Height(50).Children(func() {
		ui.Box(c).Width(44).FillHeight().Children(func() {
			for lane, name := range lanes {
				ui.Text(c, name).FontSize(10).TextColor(p.caption).SingleLine().Absolute().Left(0).Top(float32(5 + lane*14))
			}
		})
		plot := ui.Box(c).Label("Trajectory timeline").Grow(1).FillHeight().Focusable().Bind(&s.handle)
		plot.HandleInput(func(ev ui.InputEvent) bool {
			// Span bars are plain buttons (no input handler); when a primary
			// press lands on one, decline it so the engine's generic press
			// path delivers the click to the button.
			if ev.Kind == ui.InputPointerDown && ev.Button == 0 && trajSpanHitAt(model, ev.X, ev.Y, s.handle.Bounds(c).W) {
				return false
			}
			return s.plotInput(c, ev, full, model)
		}).Draw(func(pt *ui.Painter, r ui.Rect) {
			s.plotDraw(pt, r, model, p.accent)
		}).Children(func() {
			for _, span := range model.Spans {
				if span.End < model.Start || span.Start > model.End {
					continue
				}
				left, width := float32(0), float32(0)
				if model.End > model.Start {
					left = float32((math.Max(span.Start, model.Start) - model.Start) / (model.End - model.Start) * 100)
					width = float32((math.Min(span.End, model.End) - math.Max(span.Start, model.Start)) / (model.End - model.Start) * 100)
				}
				row := span.Row
				b := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
					Label:   "Trajectory span " + row.ID,
				}).Absolute().LeftPercent(left).Top(float32(5 + span.Lane*14)).WidthPercent(width).Height(8).
					Padding(0).Radius(design.RadiusXS / 2).Background(trajTimelineColor(p, row, trajCellKind(row))).
					Tooltip(trajectoryTimingDetail(row))
				if width == 0 {
					b.Width(2)
				}
				if b.Clicked() {
					*selected = row.ID
					s.clear()
				}
			}
		})
	})
	if s.active || s.viewport.active {
		ui.Row(c).Padding(design.Sp(0.5), 0, 0, design.Sp(11)).Gap(design.Sp(2)).Children(func() {
			label := fmt.Sprintf(trajectoryText("已选 %.0f–%.0f ms · %d 项", "Selected %.0f–%.0f ms · %d records"), s.start-full.Start, s.end-full.Start, len(s.matches(rows)))
			if s.sequence {
				label = fmt.Sprintf(trajectoryText("已选 %.1f–%.1f · %d 项", "Selected %.1f–%.1f · %d records"), s.start, s.end, len(s.matches(rows)))
			}
			if !s.active {
				label = trajectoryText("拖选时间范围以聚焦记录", "Drag a time range to focus records")
			}
			ui.Text(c, label).FontSize(11).TextColor(p.muted).Grow(1)
			if s.viewport.active {
				if coss.Button(c, trajectoryText("重置缩放", "Reset zoom"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Height(18).Clicked() {
					s.viewport = trajectoryViewport{}
				}
			}
			if s.active {
				if coss.Button(c, trajectoryText("清除范围", "Clear range"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Height(18).Clicked() {
					s.clear()
				}
			}
		})
	}
	s.afterPaint(rows, list, ledger)
}

// trajSpanHitAt reports whether plot-local (x, y) falls inside a rendered
// span bar — same geometry as the Absolute span buttons above.
func trajSpanHitAt(model trajectoryTimeline, x, y, width float32) bool {
	lane := int((y - 5) / 14)
	if lane < 0 || lane > 2 || y < 5 || y >= 5+float32(lane)*14+8 {
		return false
	}
	for _, span := range model.Spans {
		if span.Lane != lane || span.End < model.Start || span.Start > model.End {
			continue
		}
		left, w := float32(0), float32(0)
		if model.End > model.Start {
			left = float32((math.Max(span.Start, model.Start) - model.Start) / (model.End - model.Start) * 100)
			w = float32((math.Min(span.End, model.End) - math.Max(span.Start, model.Start)) / (model.End - model.Start) * 100)
		}
		lx := left / 100 * width
		wx := w / 100 * width
		if wx <= 0 {
			wx = 2
		}
		if x >= lx && x <= lx+wx {
			return true
		}
	}
	return false
}

func trajectoryText(zh, en string) string {
	return i18n.Pick(zh, en)
}

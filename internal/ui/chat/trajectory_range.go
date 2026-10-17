package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
)

type trajectoryRangeState struct {
	handle                          ui.Handle
	start, end, anchor              float64
	active, dragging, pendingScroll bool
}

func (s *trajectoryRangeState) clear() { h := s.handle; *s = trajectoryRangeState{handle: h} }
func (s *trajectoryRangeState) selectRange(start, end float64) {
	if math.IsNaN(start) || math.IsNaN(end) || math.IsInf(start, 0) || math.IsInf(end, 0) {
		s.clear()
		return
	}
	s.start, s.end = math.Min(start, end), math.Max(start, end)
	s.active, s.pendingScroll = true, true
}
func (s *trajectoryRangeState) matches(rows []trajRow) map[string]bool {
	if !s.active {
		return nil
	}
	model := buildRecordedTrajectoryTimeline(rows)
	if len(model.Spans) == 0 || s.end < model.Start || s.start > model.End {
		s.clear()
		return nil
	}
	matched := map[string]bool{}
	for _, row := range rows {
		for _, span := range buildRecordedTrajectoryTimeline([]trajRow{row}).Spans {
			if span.End >= s.start && span.Start <= s.end {
				matched[row.ID] = true
			}
		}
	}
	return matched
}
func rangeTimeAt(x, width float32, model trajectoryTimeline) float64 {
	if width <= 0 {
		return model.Start
	}
	return model.Start + math.Max(0, math.Min(1, float64(x/width)))*(model.End-model.Start)
}
func paintTrajectoryRange(c *ui.Context, rows []trajRow, s *trajectoryRangeState, list *ui.ListState, accent, muted, border ui.Color) {
	model := buildRecordedTrajectoryTimeline(rows)
	if len(model.Spans) == 0 || model.End <= model.Start {
		s.clear()
		return
	}
	s.matches(rows)
	ui.Column(c).Gap(3).Children(func() {
		ui.Row(c).Children(func() {
			label := trajectoryText("拖选时间范围以聚焦记录", "Drag a time range to focus records")
			if s.active {
				label = fmt.Sprintf(trajectoryText("已选 %.0f–%.0f ms · %d 项", "Selected %.0f–%.0f ms · %d records"), s.start-model.Start, s.end-model.Start, len(s.matches(rows)))
			}
			ui.Text(c, label).FontSize(11).TextColor(muted).Grow(1)
			if s.active && ui.Button(c, trajectoryText("清除范围", "Clear range")).Clicked() {
				s.clear()
			}
		})
		track := ui.Box(c).Label("Trajectory time range").FillWidth().Height(18).Border(1, border).Focusable().Bind(&s.handle)
		track.HandleInput(func(ev ui.InputEvent) bool {
			at := rangeTimeAt(ev.X, s.handle.Bounds(c).W, model)
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button != 0 {
					return false
				}
				s.handle.Focus(c)
				s.anchor = at
				s.dragging = true
				s.selectRange(at, at)
				return true
			case ui.InputPointerMove:
				if !s.dragging {
					return false
				}
				s.selectRange(s.anchor, at)
				return true
			case ui.InputPointerUp:
				if !s.dragging {
					return false
				}
				s.selectRange(s.anchor, at)
				s.dragging = false
				return true
			case ui.InputKeyDown:
				if ev.Key == ui.KeyEscape && s.active {
					s.clear()
					return true
				}
			}
			return false
		}).Draw(func(p *ui.Painter, r ui.Rect) {
			if !s.active {
				return
			}
			width := model.End - model.Start
			left := float32(math.Max(0, math.Min(1, (s.start-model.Start)/width))) * r.W
			right := float32(math.Max(0, math.Min(1, (s.end-model.Start)/width))) * r.W
			p.Fill(ui.Rect{X: r.X + left, Y: r.Y, W: max(float32(2), right-left), H: r.H}, accent, 2)
		})
	})
	if s.pendingScroll && !s.dragging && list != nil {
		matches := s.matches(rows)
		for index, row := range rows {
			if matches[row.ID] {
				list.FollowEnd = false
				list.ScrollTo(index, ui.Start)
				break
			}
		}
		s.pendingScroll = false
	}
}

package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
)

type trajectoryRangeState struct {
	handle                          ui.Handle
	viewport                        trajectoryViewport
	sequence, panning               bool
	panX, anchorX                   float32
	focusID                         string
	panStart                        float64
	start, end, anchor              float64
	active, dragging, pendingScroll bool
}

func (s *trajectoryRangeState) clear() {
	h, v, q := s.handle, s.viewport, s.sequence
	*s = trajectoryRangeState{handle: h, viewport: v, sequence: q}
}
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
	model := s.model(rows)
	if len(model.Spans) == 0 || s.end < model.Start || s.start > model.End {
		s.clear()
		return nil
	}
	matched := map[string]bool{}
	if s.sequence {
		for _, span := range model.Spans {
			if span.End > s.start && span.Start < s.end || s.start == s.end && span.Start <= s.start && span.End > s.start {
				matched[span.Row.ID] = true
			}
		}
		return matched
	}
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
	model := s.model(rows)
	if len(model.Spans) == 0 || model.End <= model.Start {
		s.clear()
		return
	}
	s.matches(rows)
	full := model
	model = s.viewport.domain(full)
	ui.Column(c).Gap(3).Children(func() {
		ui.Row(c).Children(func() {
			label := trajectoryText("拖选时间范围以聚焦记录", "Drag a time range to focus records")
			if s.sequence {
				label = trajectoryText("拖选记录范围以聚焦；滚轮缩放，右键拖动平移", "Drag records to focus; wheel to zoom, right-drag to pan")
			} else {
				label += trajectoryText("；滚轮缩放，右键拖动平移", "; wheel to zoom, right-drag to pan")
			}
			if s.active {
				label = fmt.Sprintf(trajectoryText("已选 %.0f–%.0f ms · %d 项", "Selected %.0f–%.0f ms · %d records"), s.start-full.Start, s.end-full.Start, len(s.matches(rows)))
			}
			if s.active && s.sequence {
				label = fmt.Sprintf(trajectoryText("已选 %.1f–%.1f · %d 项", "Selected %.1f–%.1f · %d records"), s.start, s.end, len(s.matches(rows)))
			}
			ui.Text(c, label).FontSize(11).TextColor(muted).Grow(1)
			if s.viewport.active && ui.Button(c, trajectoryText("重置缩放", "Reset zoom")).Clicked() {
				s.viewport = trajectoryViewport{}
			}
			if s.active && ui.Button(c, trajectoryText("清除范围", "Clear range")).Clicked() {
				s.clear()
			}
		})
		track := ui.Box(c).Label("Trajectory time range").FillWidth().Height(18).Border(1, border).Focusable().Bind(&s.handle)
		track.HandleInput(func(ev ui.InputEvent) bool {
			at := rangeTimeAt(ev.X, s.handle.Bounds(c).W, model)
			switch ev.Kind {
			case ui.InputScroll:
				s.viewport.zoom(full, float64(ev.X/s.handle.Bounds(c).W), float64(ev.DY), s.sequence)
				return true
			case ui.InputPointerDown:
				if ev.Button == 1 && s.viewport.active {
					s.panning = true
					s.panX = ev.X
					s.panStart = model.Start
					return true
				}
				if ev.Button != 0 {
					return false
				}
				s.handle.Focus(c)
				s.anchor = at
				s.anchorX = ev.X
				s.focusID = ""
				s.dragging = true
				s.selectRange(at, at)
				return true
			case ui.InputPointerMove:
				if s.panning {
					if w := s.handle.Bounds(c).W; w > 0 {
						s.viewport.pan(full, s.panStart, float64((ev.X-s.panX)/w))
					}
					return true
				}
				if !s.dragging {
					return false
				}
				s.selectRange(s.anchor, at)
				return true
			case ui.InputPointerUp:
				if s.panning {
					s.panning = false
					if math.Abs(float64(ev.X-s.panX)) < 3 {
						s.clear()
					}
					return true
				}
				if !s.dragging {
					return false
				}
				s.commitRange(full, model, at, math.Abs(float64(ev.X-s.anchorX)) < 3)
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
			if (s.focusID != "" && row.ID == s.focusID) || (s.focusID == "" && matches[row.ID]) {
				list.FollowEnd = false
				list.ScrollTo(index, ui.Start)
				break
			}
		}
		s.pendingScroll = false
		s.focusID = ""
	}
}

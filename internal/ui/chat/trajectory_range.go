package chat

import (
	"github.com/egoist/mygo/ui"
	"math"
)

type trajectoryRangeState struct {
	handle                          ui.Handle
	viewport                        trajectoryViewport
	sequence, panning               bool
	compressed, points              bool
	panX, anchorX                   float32
	focusID                         string
	panStart                        float64
	start, end, anchor              float64
	active, dragging, pendingScroll bool
}

func (s *trajectoryRangeState) clear() {
	h, v, q, compressed, points := s.handle, s.viewport, s.sequence, s.compressed, s.points
	*s = trajectoryRangeState{handle: h, viewport: v, sequence: q, compressed: compressed, points: points}
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
	for _, span := range model.Spans {
		if span.Start > s.end || span.End < s.start {
			continue
		}
		matched[span.Row.ID] = true
		if !s.sequence && span.Row.Timing != nil {
			for _, row := range rows {
				if row.Kind != "trajectory-request" && row.Timing != nil && row.Timing.OperationID == span.Row.Timing.OperationID {
					matched[row.ID] = true
				}
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

// plotInput carries the original overview's interaction: left-drag selects a
// range, wheel zooms the viewport, right-drag pans a zoomed viewport, Escape
// resets (TrajectoryTimeline.tsx drag ranges / click-sized focus).
func (s *trajectoryRangeState) plotInput(c *ui.Context, ev ui.InputEvent, full, model trajectoryTimeline) bool {
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
}

// plotDraw overlays the active selection range inside the lane plot.
func (s *trajectoryRangeState) plotDraw(p *ui.Painter, r ui.Rect, model trajectoryTimeline, accent ui.Color) {
	if !s.active {
		return
	}
	width := model.End - model.Start
	if width <= 0 {
		return
	}
	left := float32(math.Max(0, math.Min(1, (s.start-model.Start)/width))) * r.W
	right := float32(math.Max(0, math.Min(1, (s.end-model.Start)/width))) * r.W
	p.Fill(ui.Rect{X: r.X + left, Y: r.Y, W: max(float32(2), right-left), H: r.H}, accent.Alpha(.18), 2)
}

// afterPaint performs the deferred ledger scroll a range commit requested.
func (s *trajectoryRangeState) afterPaint(rows []trajRow, list *ui.ListState, ledger []trajRow) {
	if !s.pendingScroll || s.dragging || list == nil {
		return
	}
	matches := s.matches(rows)
	visible := rows
	if len(ledger) > 0 {
		visible = ledger
	}
	for index, row := range visible {
		if (s.focusID != "" && row.ID == s.focusID) || (s.focusID == "" && matches[row.ID]) {
			list.FollowEnd = false
			list.ScrollTo(index, ui.Start)
			break
		}
	}
	s.pendingScroll = false
	s.focusID = ""
}

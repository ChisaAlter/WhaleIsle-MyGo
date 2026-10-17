package chat

import "math"

func nearestTrajectoryRow(model trajectoryTimeline, at float64) string {
	id := ""
	best := math.Inf(1)
	for _, span := range model.Spans {
		distance := math.Max(span.Start-at, math.Max(at-span.End, 0))
		if distance < best {
			id, best = span.Row.ID, distance
		}
	}
	return id
}
func (s *trajectoryRangeState) commitRange(full, domain trajectoryTimeline, at float64, click bool) {
	s.selectRange(s.anchor, at)
	if len(full.Spans) == 0 {
		s.clear()
		return
	}
	width := math.Min(math.Max(1, domain.End-domain.Start), math.Max(1, full.End-full.Start)/float64(len(full.Spans)))
	if s.end-s.start < width {
		center := (s.start + s.end) / 2
		if click {
			center = s.start
		}
		width = math.Min(full.End-full.Start, math.Max(0, width))
		s.start = math.Min(math.Max(center-width/2, full.Start), full.End-width)
		s.end = s.start + width
	}
	if click {
		s.focusID = nearestTrajectoryRow(full, math.Min(s.anchor, at))
	}
	s.dragging = false
}

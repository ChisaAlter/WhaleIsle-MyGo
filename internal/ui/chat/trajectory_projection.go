package chat

import (
	"math"
	"sort"
)

// Projection changes only displayed coordinates. Stored timestamps and detail
// durations remain the original evidence, including unknown-duration markers.
func projectTrajectoryTimeline(full trajectoryTimeline, compress, points bool) trajectoryTimeline {
	out := full
	out.Spans = append([]trajectorySpan(nil), full.Spans...)
	out.Compressed = compress
	if len(out.Spans) == 0 {
		return out
	}
	order := make([]int, len(out.Spans))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := full.Spans[order[i]], full.Spans[order[j]]
		if a.Start == b.Start {
			return a.End < b.End
		}
		return a.Start < b.Start
	})
	removed := 0.0
	covered := full.Spans[order[0]].Start
	for _, index := range order {
		raw := full.Spans[index]
		if compress && raw.Start > covered {
			removed += raw.Start - covered
		}
		out.Spans[index].Start = raw.Start - removed
		out.Spans[index].End = raw.End - removed
		if points {
			out.Spans[index].End = out.Spans[index].Start
		}
		covered = math.Max(covered, raw.End)
	}
	out.Start, out.End = out.Spans[0].Start, out.Spans[0].End
	for _, span := range out.Spans {
		out.Start = math.Min(out.Start, span.Start)
		out.End = math.Max(out.End, span.End)
	}
	return out
}
func (s *trajectoryRangeState) setProjection(sequence, compressed, points bool) {
	if s.sequence != sequence || s.compressed != compressed || s.points != points {
		s.clear()
		s.viewport = trajectoryViewport{}
		s.sequence, s.compressed, s.points = sequence, compressed, points
	}
}

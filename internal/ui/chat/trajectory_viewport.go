package chat

import "math"

// A viewport is local to this session and axis mode, never persisted as timing.
type trajectoryViewport struct {
	start, end float64
	active     bool
}

func (v *trajectoryViewport) domain(full trajectoryTimeline) trajectoryTimeline {
	if !v.active || full.End <= full.Start {
		return full
	}
	duration := math.Min(v.end-v.start, full.End-full.Start)
	if duration <= 0 {
		v.active = false
		return full
	}
	start := math.Max(full.Start, math.Min(full.End-duration, v.start))
	v.start, v.end = start, start+duration
	full.Start, full.End = v.start, v.end
	return full
}
func (v *trajectoryViewport) zoom(full trajectoryTimeline, fraction, delta float64, sequence bool) {
	current := v.domain(full)
	total := full.End - full.Start
	if total <= 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		return
	}
	minimum := 20.0
	if sequence {
		minimum = 4
	}
	duration := math.Max(math.Min(minimum, total), math.Min(total, (current.End-current.Start)*math.Exp(delta*.0015)))
	if duration >= total*.999 {
		*v = trajectoryViewport{}
		return
	}
	fraction = math.Max(0, math.Min(1, fraction))
	anchor := current.Start + fraction*(current.End-current.Start)
	v.start = math.Max(full.Start, math.Min(full.End-duration, anchor-fraction*duration))
	v.end, v.active = v.start+duration, true
}
func (v *trajectoryViewport) pan(full trajectoryTimeline, anchorStart, fraction float64) {
	if !v.active {
		return
	}
	current := v.domain(full)
	duration := current.End - current.Start
	v.start = math.Max(full.Start, math.Min(full.End-duration, anchorStart-fraction*duration))
	v.end = v.start + duration
}
func buildSequenceTrajectoryTimeline(rows []trajRow) trajectoryTimeline {
	var model trajectoryTimeline
	for _, row := range rows {
		if row.Kind == "trajectory-request" {
			continue
		}
		start := float64(len(model.Spans))
		model.Spans = append(model.Spans, trajectorySpan{Row: row, Start: start, End: start + 1, DurationKnown: true})
	}
	model.End = float64(len(model.Spans))
	return model
}
func (s *trajectoryRangeState) model(rows []trajRow) trajectoryTimeline {
	if s.sequence {
		return buildSequenceTrajectoryTimeline(rows)
	}
	return projectTrajectoryTimeline(buildRecordedTrajectoryTimeline(rows), s.compressed, s.points)
}
func (s *trajectoryRangeState) setSequence(sequence bool) { s.setProjection(sequence, false, false) }

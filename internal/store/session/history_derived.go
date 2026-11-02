package session

import (
	"bytes"
	"math"
	"strings"

	schedstore "whaleisle/internal/store/schedule"
)

// HistoryDerivedMemo belongs to one UI owner. Its zero value is ready to use;
// callers must serialize access (normally on the UI thread). The pure derivation
// functions remain independent APIs. No result or source is shared globally.
//
// It retains one exact ordered source, bounded by 4 MiB including a conservative
// metadata allowance and by 8192 rows. Oversized or NaN-bearing inputs are
// derived directly and leave a useful smaller source warm. Raw bytes and timing
// values are owned, so in-place input edits cannot change the remembered source.
// Performance is a value; schedule results are independently cloned on every read.
type HistoryDerivedMemo struct {
	binding, sessionID string
	source             []historyDerivedSource
	sourceBytes        int
	ready, nilMessages bool
	performanceReady   bool
	performance        historyRecordedPerformance
	scheduleReady      bool
	cards              map[float64][]schedstore.Entry
}

const historyDerivedByteLimit = 4 << 20
const historyDerivedRowLimit = 8192
const historyDerivedRowBytes = 128

type historyRecordedPerformance struct {
	tokens, millis float64
	steps          int
}

type historyDerivedSource struct {
	seq                 float64
	streaming           bool
	raw                 string
	timingPresent       bool
	operationID         string
	started, first, end historyDerivedTime
	closed, invalid     bool
}

type historyDerivedTime struct {
	value   float64
	present bool
}

func historyTimeValue(p *float64) historyDerivedTime {
	if p == nil {
		return historyDerivedTime{}
	}
	return historyDerivedTime{value: *p, present: true}
}

func sameHistoryTime(old historyDerivedTime, current *float64) bool {
	return current == nil && !old.present || current != nil && old.present && old.value == *current
}

func historyTimingNaN(t *TrajectoryTiming) bool {
	return t != nil && (t.StartedAt != nil && math.IsNaN(*t.StartedAt) || t.FirstTokenAt != nil && math.IsNaN(*t.FirstTokenAt) || t.CompletedAt != nil && math.IsNaN(*t.CompletedAt))
}

func (old historyDerivedSource) matches(m Message) bool {
	if old.seq != m.Seq || old.streaming != m.Streaming || !bytes.Equal([]byte(old.raw), m.Raw) || old.timingPresent != (m.TrajectoryTime != nil) {
		return false
	}
	if t := m.TrajectoryTime; t != nil {
		return old.operationID == t.OperationID && old.closed == t.Closed && old.invalid == t.Invalid && sameHistoryTime(old.started, t.StartedAt) && sameHistoryTime(old.first, t.FirstTokenAt) && sameHistoryTime(old.end, t.CompletedAt)
	}
	return true
}

// prepare compares actual values and every Raw byte, not IDs, lengths, hashes or
// pointer identities. SelectedID and the accepted history binding isolate owners.
func (memo *HistoryDerivedMemo) prepare(v ViewState) bool {
	if len(v.Messages) > historyDerivedRowLimit || len(v.PlanHistoryBinding) > historyDerivedByteLimit || len(v.SelectedID) > historyDerivedByteLimit-len(v.PlanHistoryBinding) {
		return false
	}
	size := len(v.PlanHistoryBinding) + len(v.SelectedID)
	same := memo.ready && memo.binding == v.PlanHistoryBinding && memo.sessionID == v.SelectedID && len(memo.source) == len(v.Messages) && memo.nilMessages == (v.Messages == nil)
	for i, m := range v.Messages {
		if math.IsNaN(m.Seq) || historyTimingNaN(m.TrajectoryTime) {
			return false
		}
		if historyDerivedRowBytes > historyDerivedByteLimit-size {
			return false
		}
		size += historyDerivedRowBytes
		if len(m.Raw) > historyDerivedByteLimit-size {
			return false
		}
		size += len(m.Raw)
		if m.TrajectoryTime != nil {
			if len(m.TrajectoryTime.OperationID) > historyDerivedByteLimit-size {
				return false
			}
			size += len(m.TrajectoryTime.OperationID)
		}
		if same && !memo.source[i].matches(m) {
			same = false
		}
	}
	if same {
		return true
	}
	source := make([]historyDerivedSource, len(v.Messages))
	for i, m := range v.Messages {
		row := historyDerivedSource{seq: m.Seq, streaming: m.Streaming, raw: string(m.Raw), timingPresent: m.TrajectoryTime != nil}
		if t := m.TrajectoryTime; t != nil {
			row.operationID, row.closed, row.invalid = strings.Clone(t.OperationID), t.Closed, t.Invalid
			row.started, row.first, row.end = historyTimeValue(t.StartedAt), historyTimeValue(t.FirstTokenAt), historyTimeValue(t.CompletedAt)
		}
		source[i] = row
	}
	*memo = HistoryDerivedMemo{binding: strings.Clone(v.PlanHistoryBinding), sessionID: strings.Clone(v.SelectedID), source: source, sourceBytes: size, ready: true, nilMessages: v.Messages == nil}
	return true
}

// RecordedPerformance returns the unchanged pure function's loaded-window facts.
func (memo *HistoryDerivedMemo) RecordedPerformance(v ViewState) (tokens, decodeMillis float64, steps int) {
	if !memo.prepare(v) {
		return RecordedPerformance(v.Messages)
	}
	if !memo.performanceReady {
		memo.performance.tokens, memo.performance.millis, memo.performance.steps = RecordedPerformance(v.Messages)
		memo.performanceReady = true
	}
	return memo.performance.tokens, memo.performance.millis, memo.performance.steps
}

// ScheduleCards returns an independent map, slices and receipt Raw bytes. A UI's
// live-catalog overlays or a caller's edits cannot mutate the remembered receipt.
func (memo *HistoryDerivedMemo) ScheduleCards(v ViewState) map[float64][]schedstore.Entry {
	if !memo.prepare(v) {
		return ScheduleCards(v.Messages, v.SelectedID)
	}
	if !memo.scheduleReady {
		memo.cards = ScheduleCards(v.Messages, v.SelectedID)
		memo.scheduleReady = true
	}
	return cloneHistoryScheduleCards(memo.cards)
}

func cloneHistoryScheduleCards(source map[float64][]schedstore.Entry) map[float64][]schedstore.Entry {
	if source == nil {
		return nil
	}
	out := make(map[float64][]schedstore.Entry, len(source))
	for seq, rows := range source {
		if rows == nil {
			out[seq] = nil
			continue
		}
		cloned := make([]schedstore.Entry, len(rows))
		copy(cloned, rows)
		for i := range cloned {
			cloned[i].Raw = bytes.Clone(rows[i].Raw)
		}
		out[seq] = cloned
	}
	return out
}

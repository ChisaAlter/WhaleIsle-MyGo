package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	schedstore "whaleisle/internal/store/schedule"
)

func historyDerivedNumber(v float64) *float64 { return &v }

func historyDerivedFixture() ViewState {
	v := ViewState{SelectedID: "memo-session", PlanHistoryBinding: "host-a/follow-1"}
	appendEvent := func(raw string) {
		v.Messages = append(v.Messages, Message{Seq: float64(len(v.Messages) + 1), Raw: json.RawMessage(raw)})
	}
	appendEvent(`{"type":"turn/start","seq":1,"data":{"turn":1}}`)
	appendEvent(`{"type":"tool/call","seq":2,"data":{"turn":1,"callId":"schedule-one","name":"schedule_create"}}`)
	receipt := `{"id":"task-one","title":"Morning task","prompt":"Check task status","kind":"after","scheduledAt":"2030-01-02T03:04:05Z","afterSeconds":60}`
	result, _ := json.Marshal(map[string]any{"type": "tool/result", "seq": 3, "data": map[string]any{"turn": 1, "message": map[string]any{"source": map[string]string{"callId": "schedule-one"}, "content": []map[string]string{{"type": "text", "text": receipt}}}}})
	appendEvent(string(result))
	appendEvent(`{"type":"assistant/message","seq":4,"data":{"turn":1,"usage":{"outputTokens":120}}}`)
	v.Messages[3].TrajectoryTime = &TrajectoryTiming{OperationID: "attempt-one", StartedAt: historyDerivedNumber(500), FirstTokenAt: historyDerivedNumber(1000), CompletedAt: historyDerivedNumber(1600), Closed: true}
	appendEvent(`{"type":"turn/end","seq":5,"data":{"turn":1}}`)
	appendEvent(`{"type":"assistant/message","seq":6,"data":{"turn":2,"stream":[{"type":"usage","usage":{"outputTokens":80}}]}}`)
	v.Messages[5].TrajectoryTime = &TrajectoryTiming{FirstTokenAt: historyDerivedNumber(2000), CompletedAt: historyDerivedNumber(2400)}
	return v
}

func cloneHistoryDerivedFixture(v ViewState) ViewState {
	out := v
	if v.Messages == nil {
		return out
	}
	out.Messages = append([]Message{}, v.Messages...)
	for i := range out.Messages {
		out.Messages[i].Raw = bytes.Clone(v.Messages[i].Raw)
		if t := v.Messages[i].TrajectoryTime; t != nil {
			copy := *t
			if t.StartedAt != nil {
				copy.StartedAt = historyDerivedNumber(*t.StartedAt)
			}
			if t.FirstTokenAt != nil {
				copy.FirstTokenAt = historyDerivedNumber(*t.FirstTokenAt)
			}
			if t.CompletedAt != nil {
				copy.CompletedAt = historyDerivedNumber(*t.CompletedAt)
			}
			out.Messages[i].TrajectoryTime = &copy
		}
	}
	return out
}

func sameHistoryDerivedFloat(a, b float64) bool { return a == b || math.IsNaN(a) && math.IsNaN(b) }

func assertHistoryDerivedOracle(t *testing.T, memo *HistoryDerivedMemo, v ViewState) {
	t.Helper()
	wantTokens, wantMS, wantSteps := RecordedPerformance(v.Messages)
	gotTokens, gotMS, gotSteps := memo.RecordedPerformance(v)
	if !sameHistoryDerivedFloat(gotTokens, wantTokens) || !sameHistoryDerivedFloat(gotMS, wantMS) || gotSteps != wantSteps {
		t.Fatalf("performance got=%v/%v/%d want=%v/%v/%d", gotTokens, gotMS, gotSteps, wantTokens, wantMS, wantSteps)
	}
	wantCards := ScheduleCards(v.Messages, v.SelectedID)
	if got := memo.ScheduleCards(v); !reflect.DeepEqual(got, wantCards) {
		t.Fatalf("schedule got=%+v want=%+v", got, wantCards)
	}
}

func TestHistoryDerivedMemoMatchesOracleAcrossSourceChanges(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*ViewState)
	}{
		{"raw-equal-length", func(v *ViewState) {
			v.Messages[3].Raw = bytes.Replace(v.Messages[3].Raw, []byte("120"), []byte("360"), 1)
		}},
		{"raw-in-place", func(v *ViewState) {
			raw := v.Messages[3].Raw
			at := bytes.Index(raw, []byte("120"))
			copy(raw[at:at+3], "240")
		}},
		{"raw-longer", func(v *ViewState) {
			v.Messages[3].Raw = bytes.Replace(v.Messages[3].Raw, []byte("120"), []byte("1200"), 1)
		}},
		{"raw-shorter", func(v *ViewState) {
			v.Messages[3].Raw = bytes.Replace(v.Messages[3].Raw, []byte("120"), []byte("20"), 1)
		}},
		{"raw-invalid", func(v *ViewState) { v.Messages[3].Raw = []byte(`broken`) }},
		{"raw-nil", func(v *ViewState) { v.Messages[3].Raw = nil }},
		{"raw-empty", func(v *ViewState) { v.Messages[3].Raw = []byte{} }},
		{"streaming", func(v *ViewState) { v.Messages[3].Streaming = true }},
		{"seq-duplicate", func(v *ViewState) { v.Messages[5].Seq = v.Messages[3].Seq }},
		{"seq-zero", func(v *ViewState) { v.Messages[3].Seq = 0 }},
		{"reorder", func(v *ViewState) { v.Messages[3], v.Messages[5] = v.Messages[5], v.Messages[3] }},
		{"slice-range", func(v *ViewState) { v.Messages = v.Messages[1:5] }},
		{"append", func(v *ViewState) { copy := v.Messages[3]; copy.Seq = 88; v.Messages = append(v.Messages, copy) }},
		{"truncate", func(v *ViewState) { v.Messages = v.Messages[:3] }},
		{"timing-nil", func(v *ViewState) { v.Messages[3].TrajectoryTime = nil }},
		{"first-nil", func(v *ViewState) { v.Messages[3].TrajectoryTime.FirstTokenAt = nil }},
		{"end-nil", func(v *ViewState) { v.Messages[3].TrajectoryTime.CompletedAt = nil }},
		{"first-in-place", func(v *ViewState) { *v.Messages[3].TrajectoryTime.FirstTokenAt = 1100 }},
		{"end-in-place", func(v *ViewState) { *v.Messages[3].TrajectoryTime.CompletedAt = 1800 }},
		{"invalid-in-place", func(v *ViewState) { v.Messages[3].TrajectoryTime.Invalid = true }},
		{"equal-ends", func(v *ViewState) {
			*v.Messages[3].TrajectoryTime.CompletedAt = *v.Messages[3].TrajectoryTime.FirstTokenAt
		}},
		{"reverse-ends", func(v *ViewState) { *v.Messages[3].TrajectoryTime.CompletedAt = 100 }},
		{"start-in-place", func(v *ViewState) { *v.Messages[3].TrajectoryTime.StartedAt = 600 }},
		{"closed-in-place", func(v *ViewState) { v.Messages[3].TrajectoryTime.Closed = false }},
		{"operation-in-place", func(v *ViewState) { v.Messages[3].TrajectoryTime.OperationID = "attempt-two" }},
		{"selection", func(v *ViewState) { v.SelectedID = "another-session" }},
		{"same-session-new-binding", func(v *ViewState) { v.PlanHistoryBinding = "host-b/follow-1" }},
		{"schedule-call-streaming", func(v *ViewState) { v.Messages[1].Streaming = true }},
		{"schedule-end-in-place", func(v *ViewState) {
			raw := v.Messages[4].Raw
			at := bytes.Index(raw, []byte(`"seq":5`))
			raw[at+6] = '2'
		}},
		{"schedule-result-error", func(v *ViewState) {
			v.Messages[2].Raw = bytes.Replace(v.Messages[2].Raw, []byte(`"message":{`), []byte(`"message":{"isError":true,`), 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := historyDerivedFixture()
			var memo HistoryDerivedMemo
			assertHistoryDerivedOracle(t, &memo, v)
			tc.edit(&v)
			assertHistoryDerivedOracle(t, &memo, v)
			assertHistoryDerivedOracle(t, &memo, cloneHistoryDerivedFixture(v))
		})
	}
}

func TestHistoryDerivedMemoPreservesDuplicatesOrderAndNumericEdges(t *testing.T) {
	for _, seq := range []float64{3, 0, math.Copysign(0, -1), -2, math.Inf(1), math.Inf(-1), math.NaN()} {
		t.Run(fmt.Sprint(seq), func(t *testing.T) {
			v := historyDerivedFixture()
			v.Messages = v.Messages[3:4]
			v.Messages[0].Seq = seq
			duplicate := v.Messages[0]
			duplicate.Raw = bytes.Replace(duplicate.Raw, []byte("120"), []byte("999"), 1)
			v.Messages = append(v.Messages, duplicate)
			var memo HistoryDerivedMemo
			assertHistoryDerivedOracle(t, &memo, v)
			v.Messages[0].Raw = json.RawMessage(`broken`)
			assertHistoryDerivedOracle(t, &memo, v) // Invalid first duplicate must not suppress the valid row.
			v.Messages[0].Raw = duplicate.Raw
			v.Messages[0].Streaming = true
			assertHistoryDerivedOracle(t, &memo, v)
		})
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1)} {
		v := historyDerivedFixture()
		*v.Messages[3].TrajectoryTime.CompletedAt = value
		var memo HistoryDerivedMemo
		assertHistoryDerivedOracle(t, &memo, v)
	}
}

func TestHistoryDerivedMemoNilEmptyAndIndependentScheduleResults(t *testing.T) {
	var memo HistoryDerivedMemo
	for _, messages := range [][]Message{nil, {}, nil, historyDerivedFixture().Messages, {}} {
		assertHistoryDerivedOracle(t, &memo, ViewState{Messages: messages})
	}
	v := historyDerivedFixture()
	first := memo.ScheduleCards(v)
	if len(first[5]) != 1 {
		t.Fatal("fixture receipt absent")
	}
	first[5][0].Title = "mutated"
	first[5][0].Raw[0] = '!'
	first[5] = append(first[5], schedstore.Entry{ID: "injected"})
	delete(first, 5)
	first[42] = []schedstore.Entry{{ID: "other"}}
	assertHistoryDerivedOracle(t, &memo, v)
	second := memo.ScheduleCards(v)
	second[5][0].Raw[1] = '!'
	assertHistoryDerivedOracle(t, &memo, v)
	if cloneHistoryScheduleCards(nil) != nil {
		t.Fatal("nil map changed")
	}
	cloned := cloneHistoryScheduleCards(map[float64][]schedstore.Entry{1: nil, 2: {}})
	if cloned[1] != nil || cloned[2] == nil {
		t.Fatal("nil/empty receipt slices changed")
	}
}

func TestHistoryDerivedMemoBoundsKeepSmallSourceWarm(t *testing.T) {
	small := historyDerivedFixture()
	var memo HistoryDerivedMemo
	assertHistoryDerivedOracle(t, &memo, small)
	ownedSource := memo.source
	if memo.sourceBytes > historyDerivedByteLimit || len(memo.source) > historyDerivedRowLimit {
		t.Fatal("small source exceeds bound")
	}
	for _, large := range []ViewState{
		{SelectedID: "large", Messages: []Message{{Raw: json.RawMessage(strings.Repeat(" ", historyDerivedByteLimit+1))}}},
		{SelectedID: "many", Messages: make([]Message, historyDerivedRowLimit+1)},
		{SelectedID: strings.Repeat("x", historyDerivedByteLimit+1)},
		{Messages: []Message{{Seq: math.NaN()}}},
		{Messages: []Message{{TrajectoryTime: &TrajectoryTiming{FirstTokenAt: historyDerivedNumber(math.NaN())}}}},
	} {
		assertHistoryDerivedOracle(t, &memo, large)
		if len(memo.source) != len(ownedSource) || &memo.source[0] != &ownedSource[0] || memo.sessionID != small.SelectedID {
			t.Fatal("oversized or NaN input evicted small source")
		}
		assertHistoryDerivedOracle(t, &memo, small)
	}
}

func TestHistoryDerivedMemoLazyAndExactIndependentOwnership(t *testing.T) {
	v := historyDerivedFixture()
	var memo HistoryDerivedMemo
	_ = memo.ScheduleCards(v)
	if memo.performanceReady || !memo.scheduleReady {
		t.Fatal("schedule eagerly computed performance")
	}
	memo.RecordedPerformance(v)
	original := memo.source
	copy := cloneHistoryDerivedFixture(v)
	assertHistoryDerivedOracle(t, &memo, copy)
	if &memo.source[0] != &original[0] {
		t.Fatal("equal replacement missed warm cache")
	}
	copy.Messages[3].Raw[0] = '!'
	*copy.Messages[3].TrajectoryTime.CompletedAt = 9999
	if memo.source[3].raw[0] != '{' || memo.source[3].end.value != 1600 {
		t.Fatal("memo retained caller's mutable bytes or pointers")
	}
	assertHistoryDerivedOracle(t, &memo, copy)
	v = historyDerivedFixture()
	v.PlanHistoryBinding = "replacement-host"
	_ = memo.ScheduleCards(v)
	if memo.performanceReady {
		t.Fatal("changed binding reused old performance")
	}
}

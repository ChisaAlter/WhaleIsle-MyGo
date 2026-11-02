package interaction

import (
	"encoding/json"
	"testing"
)

// The memo answers exactly as the decoder does, per exact input, however the
// calls interleave and after it starts over at its size limit.
func TestProjectedPendingStatusMemoMatchesDecoder(t *testing.T) {
	q := json.RawMessage(`{"active":[{"callId":"q","state":"continued","questions":[{"id":"q","question":"Pick"}]}],"settled":[]}`)
	open := json.RawMessage(`{"active":[{"callId":"o","state":"open","questions":[{"id":"o","question":"Pick"}]}],"settled":[]}`)
	empty := json.RawMessage(`{"next-step":[],"next-turn":[]}`)
	queued := json.RawMessage(`{"next-step":[{"source":{"kind":"user-question-reply","callId":"q"}}],"next-turn":[]}`)
	inputs := [][2]json.RawMessage{{q, nil}, {q, empty}, {q, queued}, {open, nil}, {nil, nil}, {json.RawMessage(`{`), empty}}
	for round := 0; round < 3; round++ {
		for _, in := range inputs {
			if got, want := ProjectedPendingStatus(in[0], in[1]), projectedPendingStatus(in[0], in[1], nil); got != want {
				t.Fatalf("round %d %s / %s: %q, want %q", round, in[0], in[1], got, want)
			}
		}
	}
	for i := 0; i < projectedStatusMemoLimit+5; i++ {
		ProjectedPendingStatus(json.RawMessage(`{"active":[],"settled":[],"n":`+string(rune('0'+i%10))+`}`), json.RawMessage(string(rune('a'+i%26))))
	}
	if ProjectedPendingStatus(q, empty) != "question" || ProjectedPendingStatus(q, queued) != "" {
		t.Fatal("memo answered wrongly after starting over")
	}
}

package interaction

import (
	"encoding/json"
	"testing"
)

// Without a projection the history pairs ask_user_question calls with their
// transcript results: results may precede the call, later ones win, error and
// unrelated results never pair, and PTC sub-calls pair by sub-call id.
func TestQuestionHistoryPairsTranscriptResults(t *testing.T) {
	events := []json.RawMessage{
		json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"call-1","content":[{"type":"text","text":"{\"answers\":[{\"id\":\"q\",\"selected\":[\"A\"]}]}"}]}}}`),
		json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"other","content":[{"type":"text","text":"{\"answers\":[{\"id\":\"q\",\"selected\":[\"Z\"]}]}"}]}}}`),
		historyCall,
		json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"call-1","content":[{"type":"text","text":"{\"answers\":[{\"id\":\"q\",\"selected\":[\"B\"]}]}"}]}}}`),
		json.RawMessage(`{"type":"tool/result","data":{"message":{"toolCallId":"call-1","isError":true,"content":[{"type":"text","text":"{\"answers\":[{\"id\":\"q\",\"selected\":[\"A\"]}]}"}]}}}`),
		json.RawMessage(`{"type":"tool/ptc-dispatch-start","data":{"subCallId":"sub-2","name":"ask_user_question","arguments":"{\"questions\":[{\"id\":\"q\",\"question\":\"Pick\",\"options\":[{\"label\":\"A\"},{\"label\":\"B\"}]}]}"}}`),
		json.RawMessage(`{"type":"tool/ptc-dispatch","data":{"subCallId":"sub-2","content":[{"type":"text","text":"{\"answers\":[{\"id\":\"q\",\"selected\":[\"A\"]}]}"}]}}`),
	}
	s := New()
	s.SyncQuestionHistory("A", events, nil, nil, 0, 0)
	v := s.SnapshotForAgent("A").QuestionHistory
	if len(v) != 2 {
		t.Fatalf("history entries: %+v", v)
	}
	if v[0].CallID != "call-1" || v[0].Rows[0].Selected[0] != "B" {
		t.Fatalf("call-1 paired with the wrong result: %+v", v[0])
	}
	if v[1].CallID != "sub-2" || v[1].Rows[0].Selected[0] != "A" {
		t.Fatalf("sub-call paired with the wrong result: %+v", v[1])
	}
}

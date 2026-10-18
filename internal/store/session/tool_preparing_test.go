package session

import (
	"encoding/json"
	"testing"
)

func TestToolPreparingUsesOnlyExplicitNamedCallIdentity(t *testing.T) {
	var state toolPreparingState
	if foldToolPreparingChunk(&state, json.RawMessage(`{"type":"block-start","index":0,"blockType":"tool-call"}`), 1) {
		t.Fatal("nameless start became a tool")
	}
	if foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"name":"bash","argumentsDelta":"a"}`), 1) {
		t.Fatal("missing id inferred")
	}
	if foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","argumentsDelta":"prefix"}`), 1) {
		t.Fatal("unnamed delta opened a call")
	}
	if !foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","name":"bash","argumentsDelta":"{\"command\":"}`), 2) {
		t.Fatal("named delta not shown")
	}
	foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","argumentsDelta":"\"ls\"}"}`), 3)
	row := appendToolPreparingMessages(nil, state, 10)[0]
	if row.Tool.CallID != "c" || row.Tool.State != "preparing" || row.Tool.Arguments != `{"command":"ls"}` || row.ID != "tool:c" || row.TurnSeq != 10 || !row.Streaming {
		t.Fatalf("preparing: %#v", row)
	}
	foldToolPreparingChunk(&state, json.RawMessage(`{"type":"block-end","index":0,"block":{"type":"tool-call","id":"c","name":"pwsh","arguments":"settled"}}`), 4)
	if foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","argumentsDelta":"late"}`), 5) {
		t.Fatal("sealed args changed")
	}
	if state.calls["c"].args != "settled" || state.calls["c"].name != "pwsh" {
		t.Fatal("block end did not settle exact call")
	}
}

func TestToolPreparingDurableHandoffAndAttemptLifetime(t *testing.T) {
	var state toolPreparingState
	foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","name":"read","argumentsDelta":"same"}`), 1)
	s := Store{editTurnSeq: 10}
	s.assistantStream.active = &assistantStreamAttempt{AttemptID: "a", Turn: 1, Step: 0, StartedAfterSeq: 9, tools: state}
	ordinary := Message{ID: "text", Kind: MessageKindText, Role: "assistant", Body: "same", Streaming: true}
	rows := s.projectPreparingToolsLocked([]Message{ordinary})
	if len(rows) != 2 || rows[0].ID != "text" {
		t.Fatal("tool changed text/reasoning projection")
	}
	durable := toolLifecycleEvent("tool/call", 12, `{"callId":"c","name":"read","arguments":"same"}`)
	rows = s.projectPreparingToolsLocked([]Message{ordinary, durable})
	if len(rows) != 2 || rows[1].Tool != nil {
		t.Fatal("preparing duplicated durable call")
	}
	unrelated := toolLifecycleEvent("tool/call", 12, `{"callId":"other","name":"read","arguments":"same"}`)
	if len(s.projectPreparingToolsLocked([]Message{unrelated})) != 2 {
		t.Fatal("text/name-based suppression")
	}
	s.assistantStream.retained = s.assistantStream.active
	s.assistantStream.active = nil
	if len(s.projectPreparingToolsLocked(nil)) != 1 {
		t.Fatal("committed handoff gap lost preparing")
	}
	s.assistantStream.retained = nil
	if len(s.projectPreparingToolsLocked(nil)) != 0 {
		t.Fatal("abandoned/retired attempt left preparing")
	}
}

func TestToolPreparingClosedTurnAndMalformedEnd(t *testing.T) {
	var state toolPreparingState
	foldToolPreparingChunk(&state, json.RawMessage(`{"type":"tool-call-delta","index":0,"id":"c","name":"read","argumentsDelta":"x"}`), 1)
	if foldToolPreparingChunk(&state, json.RawMessage(`{"type":"block-end","index":0,"block":{"type":"tool-call","id":"different","name":"read","arguments":"wrong"}}`), 2) {
		t.Fatal("block end guessed identity")
	}
	s := Store{}
	s.assistantStream.active = &assistantStreamAttempt{Turn: 1, Step: 2, StartedAfterSeq: 5, tools: state}
	end := Message{Raw: json.RawMessage(`{"type":"turn/end","seq":10,"data":{"turn":1}}`)}
	if len(s.projectPreparingToolsLocked([]Message{end})) != 1 {
		t.Fatal("closed turn retained preparing")
	}
	unrelated := Message{Raw: json.RawMessage(`{"type":"turn/end","seq":10,"data":{"turn":2}}`)}
	if len(s.projectPreparingToolsLocked([]Message{unrelated})) != 2 {
		t.Fatal("unrelated turn suppressed preparing")
	}
}

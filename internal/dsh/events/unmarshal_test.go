package events

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestUnmarshalEventRegistryCoversEventsByType(t *testing.T) {
	for typ, info := range EventsByType {
		if _, ok := goTypeReflect[info.GoType]; !ok {
			t.Fatalf("EventsByType[%q] GoType %q missing from goTypeReflect", typ, info.GoType)
		}
	}
}

func TestUnmarshalEventToolCall(t *testing.T) {
	raw := []byte(`{
		"type": "tool/call",
		"seq": 4,
		"time": 100,
		"data": {
			"name": "read",
			"arguments": "{\"path\":\"README.md\"}",
			"callId": "call_abc",
			"step": 1,
			"turn": 2
		}
	}`)
	v, err := UnmarshalEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	tc, ok := v.(ToolCall)
	if !ok {
		t.Fatalf("got %T", v)
	}
	if tc.Type != TypeToolCall || tc.Seq != 4 || tc.Data.Name != "read" || tc.Data.CallId != "call_abc" {
		t.Fatalf("%+v", tc)
	}
}

func TestUnmarshalEventAssistantAttemptStream(t *testing.T) {
	raw := []byte(`{
		"type": "assistant/attempt",
		"seq": 5,
		"time": 200,
		"data": {
			"turn": 1,
			"step": 0,
			"stream": [{"type":"text-delta","text":"hel"}]
		}
	}`)
	v, err := UnmarshalEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	aa, ok := v.(AssistantAttempt)
	if !ok {
		t.Fatalf("got %T", v)
	}
	if len(aa.Data.Stream) != 1 || string(aa.Data.Stream[0]) == "" {
		t.Fatalf("stream = %v", aa.Data.Stream)
	}
}

func TestUnmarshalEventAssistantMessage(t *testing.T) {
	raw := []byte(`{
		"type": "assistant/message",
		"seq": 6,
		"time": 300,
		"surfaceOp": "append",
		"data": {
			"turn": 1,
			"step": 0,
			"stream": [],
			"message": {
				"id": "msg_asst_1",
				"role": "assistant",
				"content": [{"type":"text","text":"Hello"}],
				"source": {"kind":"model","model":"m","provider":"p"}
			}
		}
	}`)
	v, err := UnmarshalEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	am, ok := v.(AssistantMessage)
	if !ok {
		t.Fatalf("got %T", v)
	}
	if am.Data.Message.Id != "msg_asst_1" || len(am.Data.Message.Content) != 1 {
		t.Fatalf("%+v", am)
	}
}

func TestUnmarshalEventApprovalAsked(t *testing.T) {
	raw := []byte(`{
		"type": "approval/asked",
		"seq": 7,
		"time": 400,
		"data": {
			"id": "appr_1",
			"toolName": "bash",
			"callId": "c1",
			"reason": "run ls"
		}
	}`)
	v, err := UnmarshalEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	aa, ok := v.(ApprovalAsked)
	if !ok {
		t.Fatalf("got %T", v)
	}
	if aa.Data.ToolName != "bash" || aa.Data.Id != "appr_1" {
		t.Fatalf("%+v", aa)
	}
}

func TestUnmarshalEventUnknownType(t *testing.T) {
	_, err := UnmarshalEvent([]byte(`{"type":"assistant/thinking","seq":1,"data":{}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownEventType) {
		t.Fatalf("err = %v", err)
	}
}

func TestUnmarshalEventInvalidJSON(t *testing.T) {
	_, err := UnmarshalEvent([]byte(`{`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnmarshalEventRoundTripViaMarshal(t *testing.T) {
	in := ToolCall{
		Type: TypeToolCall,
		Seq:  99,
		Time: 1,
		Data: ToolCallData{Name: "grep", Arguments: "{}", CallId: "x", Step: 0, Turn: 0},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := UnmarshalEvent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.(ToolCall) != in {
		t.Fatalf("round-trip mismatch")
	}
}
